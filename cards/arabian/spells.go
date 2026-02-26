package arabian

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/mage/mage/pkg/mage"
	. "github.com/mage/mage/pkg/mage/core"
)

// sacrificeCreatureCaptureCMCCost sacrifices a creature and stores its CMC
// in g.CurrentX so the effect can read it via g.XValue().
type sacrificeCreatureCaptureCMCCost struct{}

func (c *sacrificeCreatureCaptureCMCCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	for _, p := range g.Battlefield {
		if p.Controller == controller && p.HasType(TypeCreature) && p.ID() != sourceID {
			return true
		}
	}
	return false
}

func (c *sacrificeCreatureCaptureCMCCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	var candidates []*Permanent
	for _, p := range g.Battlefield {
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
	g.CurrentX = chosen.Card.ManaCost().CMC()
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
	Register("Army of Allah", withExpansion(func() Card {
		return NewInstant("Army of Allah", "{1}{W}{W}",
			NewSpellAbility(BoostAllMatchingUntilEndOfTurn(Fixed(2), Fixed(0), IsAttacking)),
		)
	}))

	// Oracle: "The next time a source of your choice would deal damage to you this turn,
	// instead that source deals that much damage to you and Eye for an Eye deals that much
	// damage to that source's controller."
	// XXX: Eye for an Eye deferred — needs damage source tracking + replacement effect
	Register("Eye for an Eye", withExpansion(func() Card {
		return NewInstant("Eye for an Eye", "{W}{W}", nil)
	}))

	// Oracle: "Blocking creatures get +0/+3 until end of turn."
	Register("Piety", withExpansion(func() Card {
		return NewInstant("Piety", "{2}{W}",
			NewSpellAbility(BoostAllMatchingUntilEndOfTurn(Fixed(0), Fixed(3), IsBlocking)),
		)
	}))

	// Oracle: "Players play a Magic subgame, using their libraries as their decks.
	// Each player who doesn't win the subgame loses half their life, rounded up."
	// XXX: Shahrazad skipped — subgame mechanic, banned in all formats
	Register("Shahrazad", withExpansion(func() Card {
		return NewSorcery("Shahrazad", "{W}{W}", nil)
	}))

	// ===== GREEN SPELLS =====

	// Oracle: "Destroy target permanent."
	Register("Desert Twister", withExpansion(func() Card {
		return NewSorcery("Desert Twister", "{4}{G}{G}",
			NewTargetedSpell(TargetPermanent(), DestroyTargetPermanent()),
		)
	}))

	// Oracle: "As an additional cost to cast this spell, sacrifice a creature.
	// Add X mana of any one color, where X is 1 plus the sacrificed creature's mana value.
	// Spend this mana only to cast creature spells."
	Register("Metamorphosis", withExpansion(func() Card {
		return NewSorcery("Metamorphosis", "{G}",
			NewSpellAbility(FuncEffect("add mana equal to 1 + sacrificed creature's CMC",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g GameMutator, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					cmc := g.XValue() // captured by sacrificeCreatureCaptureCMCCost
					amount := 1 + cmc
					p := g.GetPlayer(controller)
					if p != nil && amount > 0 {
						p.ManaPool().Add(Green, amount)
						g.SetCreatureManaOnly(controller)
					}
					return nil
				})),
			WithAdditionalCost(&sacrificeCreatureCaptureCMCCost{}),
		)
	}))

	// Oracle: "Sandstorm deals 1 damage to each attacking creature."
	Register("Sandstorm", withExpansion(func() Card {
		return NewInstant("Sandstorm", "{G}",
			NewSpellAbility(DealDamageToAllCreatures(Fixed(1), IsAttacking)),
		)
	}))
}
