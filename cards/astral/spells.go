package astral

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

const whimsyActionCount = 18

func applyWhimsyAction(g *Game, sourceID, controllerID uuid.UUID, action int) error {
	switch action {
	case 0: // Boomerang
		chosen := dsl.BindPermanent("chosen permanent")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.Not(dsl.IsEnchanted)), chosen),
			dsl.ReturnToHand(dsl.From(chosen)),
		), sourceID, controllerID, nil)
	case 1: // Twiddle — untap
		chosen := dsl.BindPermanent("random permanent")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.And(dsl.IsTapped, dsl.Or(dsl.IsArtifact, dsl.IsCreature, dsl.IsLand))), chosen),
			dsl.Untap(dsl.From(chosen)),
		), sourceID, controllerID, nil)
	case 2: // Twiddle — tap
		return ApplyEffect(g, ApplyToRandomPermanent(Tap(), And(IsUntapped, Or(IsArtifact, IsCreature, IsLand))), sourceID, controllerID, nil)
	case 3: // Aladdin's Ring
		return ApplyEffect(g, ApplyToRandomDamageTarget(AladdinsRingEffect()), sourceID, controllerID, nil)
	case 4: // Ancestral Recall
		chosen := dsl.BindPlayer("random player")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPlayer(dsl.EachPlayer, chosen),
			dsl.Draw(dsl.FromPlayer(chosen), dsl.Fixed(3)),
		), sourceID, controllerID, nil)
	case 5: // Crumble
		return ApplyEffect(g, ApplyToRandomPermanent(CrumbleEffect(), IsArtifact), sourceID, controllerID, nil)
	case 6: // Disenchant
		chosen := dsl.BindPermanent("chosen permanent")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.Or(dsl.IsArtifact, dsl.IsEnchantment)), chosen),
			dsl.Destroy(dsl.From(chosen)),
		), sourceID, controllerID, nil)
	case 7: // Healing Salve — gain life
		chosen := dsl.BindPlayer("random player")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPlayer(dsl.EachPlayer, chosen),
			dsl.GainLife(dsl.FromPlayer(chosen), dsl.Fixed(3)),
		), sourceID, controllerID, nil)
	case 8: // Healing Salve — prevent damage
		return ApplyEffect(g, ApplyToRandomDamageTarget(HealingSalvePreventionEffect()), sourceID, controllerID, nil)
	case 9: // Fissure
		chosen := dsl.BindPermanent("chosen permanent")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPermanent(dsl.MatchingPermanents(dsl.Or(dsl.IsCreature, dsl.IsLand)), chosen),
			dsl.DestroyWithoutRegeneration(dsl.From(chosen)),
		), sourceID, controllerID, nil)
	case 10: // Millstone
		chosen := dsl.BindPlayer("random player")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPlayer(dsl.EachPlayer, chosen),
			dsl.Mill(dsl.FromPlayer(chosen), dsl.Fixed(2)),
		), sourceID, controllerID, nil)
	case 11: // The Hive
		return ApplyEffect(g, TheHiveEffect(), sourceID, controllerID, nil)
	case 12: // Nevinyrral's Disk
		return ApplyEffect(g, dsl.DestroyEach(dsl.MatchingPermanents(dsl.Or(dsl.IsArtifact, dsl.IsCreature, dsl.IsEnchantment))), sourceID, controllerID, nil)
	case 13: // Bottle of Suleiman
		return ApplyEffect(g, BottleOfSuleimanEffect(), sourceID, controllerID, nil)
	case 14: // Pandora's Box
		return ApplyEffect(g, PandorasBoxEffect(), sourceID, controllerID, nil)
	case 15: // Disrupting Scepter
		chosen := dsl.BindPlayer("random player")
		return ApplyEffect(g, dsl.Sequence(
			dsl.ChooseRandomPlayer(dsl.EachPlayer, chosen),
			dsl.Discard(dsl.FromPlayer(chosen), dsl.Fixed(1)),
		), sourceID, controllerID, nil)
	case 16: // Fog
		return ApplyEffect(g, PreventAllCombatDamage(), sourceID, controllerID, nil)
	case 17: // Sindbad
		return ApplyEffect(g, SindbadEffect(), sourceID, controllerID, nil)
	}
	return nil
}

func init() {
	registerSpells()
}

func registerSpells() {

	// Call from the Grave {2}{B}
	// Sorcery
	// Put a random creature from a random graveyard into play under your control. Call from the Grave deals to you an amount of damage equal to that creature's casting cost.
	Register("Call from the Grave", func() Card {
		return NewSorcery("Call from the Grave", "{2}{B}",
			NewSpellAbility(FuncEffect("return a random creature from a random graveyard and take damage equal to its mana value", EffectProperties{},
				func(g *Game, sourceID, controllerID uuid.UUID, _ []uuid.UUID) error {
					if g.PlayerCount() == 0 {
						return nil
					}
					graveyardOwner := g.PlayerAt(g.RandIntn(g.PlayerCount()))
					chosen := g.RandomCardFromGraveyard(graveyardOwner, IsCreatureCard)
					if chosen == nil {
						return nil
					}
					card, ok := g.MoveFromGraveyard(graveyardOwner.PlayerID(), chosen.ID(), ZoneBattlefield)
					if !ok {
						return nil
					}
					g.PutOnBattlefield(card, controllerID)
					g.DealDamageToPlayer(g.GetPlayer(controllerID), card.ManaCost().CMC(), sourceID)
					return nil
				})),
		)
	})

	// Orcish Catapult {X}{R}{R}
	// Instant
	// Randomly distribute X -0/-1 counters among a random number of random target creatures.
	Register("Orcish Catapult", func() Card {
		return NewInstant("Orcish Catapult", "{X}{R}{R}",
			NewSpell(
				RandomCounterDistribution(M0M1, XValue()),
				WithTarget(TargetRandomCount(TargetOneToXCreatures())),
			),
		)
	})

	// Whimsy {X}{U}{U}
	// Sorcery
	// Play X random fast effects.
	Register("Whimsy", func() Card {
		return NewSorcery("Whimsy", "{X}{U}{U}",
			NewSpellAbility(FuncEffect("perform X random actions", EffectProperties{},
				func(g *Game, sourceID, controllerID uuid.UUID, _ []uuid.UUID) error {
					for range g.XValue() {
						if err := applyWhimsyAction(g, sourceID, controllerID, g.RandIntn(whimsyActionCount)); err != nil {
							return err
						}
					}
					return nil
				})),
		)
	})

}
