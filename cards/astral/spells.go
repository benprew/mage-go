package astral

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

const whimsyActionCount = 18

func applyWhimsyAction(g *Game, sourceID, controllerID uuid.UUID, action int) error {
	switch action {
	case 0: // Boomerang
		return ApplyEffect(g, ApplyToRandomPermanent(BoomerangEffect(), Not(IsEnchanted)), sourceID, controllerID, nil)
	case 1: // Twiddle — untap
		return ApplyEffect(g, ApplyToRandomPermanent(UntapTarget(), And(IsTapped, Or(IsArtifact, IsCreature, IsLand))), sourceID, controllerID, nil)
	case 2: // Twiddle — tap
		return ApplyEffect(g, ApplyToRandomPermanent(Tap(), And(IsUntapped, Or(IsArtifact, IsCreature, IsLand))), sourceID, controllerID, nil)
	case 3: // Aladdin's Ring
		return ApplyEffect(g, ApplyToRandomDamageTarget(AladdinsRingEffect()), sourceID, controllerID, nil)
	case 4: // Ancestral Recall
		return ApplyEffect(g, ApplyToRandomPlayer(AncestralRecallEffect()), sourceID, controllerID, nil)
	case 5: // Crumble
		return ApplyEffect(g, ApplyToRandomPermanent(CrumbleEffect(), IsArtifact), sourceID, controllerID, nil)
	case 6: // Disenchant
		return ApplyEffect(g, ApplyToRandomPermanent(DisenchantEffect(), Or(IsArtifact, IsEnchantment)), sourceID, controllerID, nil)
	case 7: // Healing Salve — gain life
		return ApplyEffect(g, ApplyToRandomPlayer(HealingSalveGainEffect()), sourceID, controllerID, nil)
	case 8: // Healing Salve — prevent damage
		return ApplyEffect(g, ApplyToRandomDamageTarget(HealingSalvePreventionEffect()), sourceID, controllerID, nil)
	case 9: // Fissure
		return ApplyEffect(g, ApplyToRandomPermanent(FissureEffect(), Or(IsCreature, IsLand)), sourceID, controllerID, nil)
	case 10: // Millstone
		return ApplyEffect(g, ApplyToRandomPlayer(MillstoneEffect()), sourceID, controllerID, nil)
	case 11: // The Hive
		return ApplyEffect(g, TheHiveEffect(), sourceID, controllerID, nil)
	case 12: // Nevinyrral's Disk
		return ApplyEffect(g, NevinyrralsDiskEffect(), sourceID, controllerID, nil)
	case 13: // Bottle of Suleiman
		return ApplyEffect(g, BottleOfSuleimanEffect(), sourceID, controllerID, nil)
	case 14: // Pandora's Box
		return ApplyEffect(g, PandorasBoxEffect(), sourceID, controllerID, nil)
	case 15: // Disrupting Scepter
		return ApplyEffect(g, ApplyToRandomPlayer(DisruptingScepterEffect()), sourceID, controllerID, nil)
	case 16: // Fog
		return ApplyEffect(g, FogEffect(), sourceID, controllerID, nil)
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
