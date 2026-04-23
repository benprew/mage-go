package arabian

import (
	"fmt"

	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// sacrificeCreatureCaptureCMCCost sacrifices a creature and stores its CMC
// in g.CurrentX so the effect can read it via g.XValue().
type sacrificeCreatureCaptureCMCCost struct{}

func (c *sacrificeCreatureCaptureCMCCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.AllBattlefield() {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			return true
		}
	}
	return false
}

func (c *sacrificeCreatureCaptureCMCCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.AllBattlefield() {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no creature to sacrifice")
	}
	player := g.GetPlayer(controller)
	chosen := player.ChoosePermanent(candidates, "sacrifice creature", g)
	if chosen == nil {
		return fmt.Errorf("no creature to sacrifice")
	}
	g.SetXValue(chosen.Card.ManaCost().CMC())
	g.Sacrifice(chosen)
	return nil
}

func (c *sacrificeCreatureCaptureCMCCost) Text() string { return "Sacrifice a creature" }

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== WHITE SPELLS =====

	// Oracle: "Attacking creatures get +2/+0 until end of turn."
	Register("Army of Allah", func() Card {
		return NewInstant("Army of Allah", "{1}{W}{W}",
			NewSpellAbility(Boost(Fixed(2), Fixed(0)).Targeting(ToAllMatching(IsAttacking))),
		)
	})

	// Oracle: "The next time a source of your choice would deal damage to you this turn,
	// instead that source deals that much damage to you and Eye for an Eye deals that much
	// damage to that source's controller."
	Register("Eye for an Eye", func() Card {
		return NewInstant("Eye for an Eye", "{W}{W}",
			// TODO: convert to pipeline — needs damage reflection + choose-source primitives
			NewSpellAbility(FuncEffect("reflect next damage to source's controller",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					// Let the player choose a damage source (permanent) if any exist
					opponent := g.GetOpponent(controller)
					var candidates []*Permanent
					if opponent != nil {
						candidates = g.FilterBattlefield(ControlledBy(opponent.PlayerID()))
					}
					var chosenSourceID uuid.UUID
					if len(candidates) > 0 {
						player := g.GetPlayer(controller)
						chosen := player.ChoosePermanent(candidates, "choose damage source", g)
						if chosen != nil {
							chosenSourceID = chosen.ID()
						}
					}
					g.SetDamageReflection(controller, sourceID, chosenSourceID)
					return nil
				})),
		)
	})

	// Oracle: "Blocking creatures get +0/+3 until end of turn."
	Register("Piety", func() Card {
		return NewInstant("Piety", "{2}{W}",
			NewSpellAbility(Boost(Fixed(0), Fixed(3)).Targeting(ToAllMatching(IsBlocking))),
		)
	})

	// Oracle: "Players play a Magic subgame, using their libraries as their decks.
	// Each player who doesn't win the subgame loses half their life, rounded up."
	// UNIMPLEMENTABLE: Subgame mechanic requires recursive game instances.
	// Banned in all sanctioned formats. Registered as a no-op spell.
	Register("Shahrazad", func() Card {
		return NewSorcery("Shahrazad", "{W}{W}", nil)
	})

	// ===== GREEN SPELLS =====

	// Oracle: "Destroy target permanent."
	Register("Desert Twister", func() Card {
		return NewSorcery("Desert Twister", "{4}{G}{G}",
			NewTargetedSpell(TargetPermanent(), DestroyTargetPermanent()),
		)
	})

	// Oracle: "As an additional cost to cast this spell, sacrifice a creature.
	// Add X mana of any one color, where X is 1 plus the sacrificed creature's mana value.
	// Spend this mana only to cast creature spells."
	Register("Metamorphosis", func() Card {
		return NewSorcery("Metamorphosis", "{G}",
			// TODO: convert to pipeline — needs dynamic mana addition + color choice + creature-mana-only primitives
			NewSpellAbility(FuncEffect("add mana equal to 1 + sacrificed creature's CMC",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					cmc := g.XValue() // captured by sacrificeCreatureCaptureCMCCost
					amount := 1 + cmc
					p := g.GetPlayer(controller)
					if p != nil && amount > 0 {
						// "Add X mana of any one color" — player chooses
						color := p.ChooseManaColor("Metamorphosis: choose a color")
						p.ManaPool().Add(color, amount)
						g.SetCreatureManaOnly(controller)
					}
					return nil
				})),
			WithAdditionalCost(&sacrificeCreatureCaptureCMCCost{}),
		)
	})

	// Oracle: "Sandstorm deals 1 damage to each attacking creature."
	Register("Sandstorm", func() Card {
		return NewInstant("Sandstorm", "{G}",
			NewSpellAbility(DealDamageToAllCreatures(Fixed(1), IsAttacking)),
		)
	})
}
