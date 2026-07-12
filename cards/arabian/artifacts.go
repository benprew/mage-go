package arabian

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/catalog"
	. "github.com/benprew/mage-go/pkg/mage/dsl"
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

// pyramidsDestructionReplacement prevents the next destruction of a specific land
// and removes all damage from it instead. One-shot.
type pyramidsDestructionReplacement struct {
	permanentID uuid.UUID
	sourceID    uuid.UUID
	consumed    bool
}

func (r *pyramidsDestructionReplacement) SourceID() uuid.UUID   { return r.sourceID }
func (r *pyramidsDestructionReplacement) GetDuration() Duration { return EndOfTurn }

func (r *pyramidsDestructionReplacement) Matches(a Action, g GameReader) bool {
	da, ok := a.(*DestroyPermanentAction)
	if !ok {
		return false
	}
	return da.PermanentID() == r.permanentID
}

func (r *pyramidsDestructionReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	// Remove all damage marked on the land
	perm := g.MutablePermanent(r.permanentID)
	if perm != nil {
		perm.Damage = 0
	}
	return nil // prevent destruction
}

func (r *pyramidsDestructionReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *pyramidsDestructionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

func init() {
	registerArtifacts()
}

func registerArtifacts() {
	// Oracle: "{X}, {T}: The next time you would draw a card this turn, instead look at
	// the top X cards of your library, put all but one of them on the bottom of your
	// library in a random order, then draw a card. X can't be 0."
	Register("Aladdin's Lamp", func() Card {
		return NewArtifact("Aladdin's Lamp", "{10}",
			WithActivatedAbility(
				// TODO: convert to pipeline — needs SetDrawReplacement primitive
				FuncEffect("set up draw replacement",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
						x := g.XValue()
						if x <= 0 {
							return nil // X can't be 0
						}
						g.SetDrawReplacement(controller, x)
						return nil
					}),
				Tap(),
				WithCost(ManaCostOf("{X}")),
			),
		)
	})

	// Oracle: "{8}, {T}: Aladdin's Ring deals 4 damage to any target."
	Register("Aladdin's Ring", func() Card {
		return NewArtifact("Aladdin's Ring", "{8}",
			WithActivatedAbility(
				DealDamage(Fixed(4)),
				Tap(),
				WithCost(ManaCostOf("{8}")),
				WithTarget(TargetDamageAnyTarget()),
			),
		)
	})

	// Oracle: "{1}, Sacrifice Bottle of Suleiman: Flip a coin. If you win the flip,
	// create a 5/5 colorless Djinn artifact creature token with flying. If you lose
	// the flip, Bottle of Suleiman deals 5 damage to you."
	Register("Bottle of Suleiman", func() Card {
		return NewArtifact("Bottle of Suleiman", "{4}",
			WithActivatedAbility(
				IfElse("flip coin: 5/5 Djinn or 5 damage",
					FlipCoinCond{},
					CreateToken("Djinn", 5, 5, []CardType{TypeArtifact, TypeCreature}, []string{"Djinn"}, Flying),
					DealDamageToPlayers(Fixed(5), SelectController()),
				),
				ManaCostOf("{1}"),
				WithCost(SacrificeSourceCost()),
			),
		)
	})

	// Oracle: "Whenever one or more other nontoken permanents with a name originally
	// printed in the Arabian Nights expansion are on the battlefield, their controllers
	// sacrifice them. Players can't cast spells or play lands with a name originally
	// printed in the Arabian Nights expansion."
	Register("City in a Bottle", func() Card {
		return NewArtifact("City in a Bottle", "{2}",
			// ETB: sacrifice all other nontoken Arabian Nights permanents
			WithAbility(
				EntersBattlefieldTrigger(
					// TODO: convert to pipeline — needs ForEach with nontoken + set filter + sacrifice-excluding-self
					FuncEffect("sacrifice all other Arabian Nights nontoken permanents",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, p := range g.FilterBattlefield(PrintedInSet("ARN")) {
								if p.ID() == sourceID {
									continue
								}
								if p.IsToken {
									continue
								}
								g.DoSacrifice(p)
							}
							return nil
						}),
					false,
				),
			),
			// Whenever any nontoken Arabian Nights permanent enters (not self), sacrifice it
			WithAbility(
				NewTriggered(EvtZoneChange, false,
					// TODO: convert to pipeline — needs ForEach with nontoken + set filter + sacrifice-excluding-self
					FuncEffect("sacrifice entering Arabian Nights permanent",
						EffectProperties{Outcome: OutcomeDetriment},
						func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							for _, p := range g.FilterBattlefield(PrintedInSet("ARN")) {
								if p.ID() == sourceID {
									continue
								}
								if p.IsToken {
									continue
								}
								g.DoSacrifice(p)
							}
							return nil
						}),
				).
					SetCondition(func(evt *GameEvent, g GameReader, sourceID, _ uuid.UUID) bool {
						if evt.SourceID == sourceID {
							return false // don't trigger on self entering
						}
						perm := g.FindPermanent(evt.SourceID)
						if perm == nil {
							return false
						}
						return catalog.Global().CardInSet("ARN", perm.Name()) && !perm.IsToken
					}).AndConditionData(EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield}),
			),
			// Continuous: block casting/playing Arabian Nights cards
			WithStaticAbility(
				FuncContinuousEffect(LayerAbility, WhileOnBattlefield,
					func(g *Game, sourceID uuid.UUID) error {
						g.AddExpansionCastBlock("ARN")
						return nil
					},
				),
			),
		)
	})

	// Oracle: "{2}, {T}: Untap target attacking creature you control. Prevent all combat
	// damage that would be dealt to and dealt by that creature this turn."
	// Combat damage prevention achieved via RemoveFromCombat — once removed, no damage
	// is assigned to or by the creature.
	Register("Ebony Horse", func() Card {
		return NewArtifact("Ebony Horse", "{3}",
			WithActivatedAbility(
				Pipeline("untap and remove from combat",
					EffectProperties{Outcome: OutcomeBenefit},
					SnapshotPermanent(SelectTarget, "t"),
					UntapGathered("t"),
					RemoveFromCombatGathered("t"),
				),
				Tap(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreatureYouControl(IsAttacking)),
			),
		)
	})

	// Oracle: "{2}, {T}: Target creature gains flying until end of turn."
	Register("Flying Carpet", func() Card {
		return NewArtifact("Flying Carpet", "{4}",
			WithActivatedAbility(
				GrantKeyword(Flying),
				Tap(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Oracle: "{2}, {T}, Discard the last card you drew this turn: Draw a card."
	Register("Jandor's Ring", func() Card {
		return NewArtifact("Jandor's Ring", "{6}",
			WithActivatedAbility(
				DrawCards(Fixed(1)),
				Tap(),
				WithCost(ManaCostOf("{2}")),
				WithCost(&discardLastDrawnCost{}),
			),
		)
	})

	// Oracle: "{3}, {T}: Untap target creature."
	Register("Jandor's Saddlebags", func() Card {
		return NewArtifact("Jandor's Saddlebags", "{2}",
			WithActivatedAbility(
				UntapTarget(),
				Tap(),
				WithCost(ManaCostOf("{3}")),
				WithTarget(TargetCreature()),
			),
		)
	})

	// Oracle: "Remove Jeweled Bird from your deck before playing if you're not playing
	// for ante. {T}: Ante Jeweled Bird. If you do, put all other cards you own from
	// the ante zone into your graveyard, then draw a card."
	Register("Jeweled Bird", func() Card {
		return NewArtifact("Jeweled Bird", "{1}",
			WithActivatedAbility(
				// TODO: convert to pipeline — needs ante zone manipulation primitives
				FuncEffect("ante Jeweled Bird, return other ante to graveyard, draw",
					EffectProperties{Outcome: OutcomeBenefit},
					func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
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
				Tap(),
			),
		)
	})

	// Oracle: "{2}: Choose one — Destroy target Aura attached to a land. / The next time
	// target land would be destroyed this turn, remove all damage marked on it instead."
	Register("Pyramids", func() Card {
		c := NewArtifact("Pyramids", "{6}",
			WithActivatedAbility(
				// TODO: convert to pipeline — needs modal with replacement effect registration
				FuncEffect("destroy aura on land or protect land from destruction",
					EffectProperties{},
					func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
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
							// Mode 2: The next time target land would be destroyed this
							// turn, remove all damage marked on it instead.
							g.AddReplacementEffect(&pyramidsDestructionReplacement{
								permanentID: targets[0],
								sourceID:    sourceID,
							})
						}
						return nil
					}),
				ManaCostOf("{2}"),
				WithTarget(TargetPermanent(Or(IsAuraOnLand, IsLand))),
			),
		)
		c.SetModes([]string{
			"Destroy target Aura attached to a land",
			"The next time target land would be destroyed this turn, remove all damage marked on it instead",
		})
		return c
	})

	// Oracle: "{5}, {T}, Exile Ring of Ma'rûf: The next time you would draw a card this
	// turn, instead put a card you own from outside the game into your hand."
	// UNIMPLEMENTABLE: Wish/sideboard mechanic requires access to cards outside
	// the game, which the engine does not model. Registered as a no-op artifact.
	Register("Ring of Ma'rûf", func() Card {
		return NewArtifact("Ring of Ma'rûf", "{5}")
	})

	// Oracle: "{2}, {T}: Target creature gains islandwalk until end of turn. When that
	// creature dies this turn, destroy Sandals of Abdallah."
	Register("Sandals of Abdallah", func() Card {
		return NewArtifact("Sandals of Abdallah", "{4}",
			WithActivatedAbility(
				Pipeline("grant islandwalk, destroy self if creature dies",
					EffectProperties{Outcome: OutcomeBenefit},
					SnapshotPermanent(SelectTarget, "t"),
					GrantKeyword(Islandwalk),
					&RegisterDelayedTriggerData{
						EventType:     EvtZoneChange,
						MatchEventVar: "t",
						MatchFromZone: ZoneBattlefield,
						MatchToZone:   ZoneGraveyard,
						Effects:       []Effect{DestroyTarget()},
					},
				),
				Tap(),
				WithCost(ManaCostOf("{2}")),
				WithTarget(TargetCreature()),
			),
		)
	})
}
