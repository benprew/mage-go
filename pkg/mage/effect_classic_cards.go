package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// AladdinsRingEffect deals 4 damage to any target.
func AladdinsRingEffect() Effect { return DealDamage(Fixed(4)) }

// BottleOfSuleimanEffect flips a coin, creating a 5/5 flying Djinn artifact
// creature token on a win or dealing 5 damage to the controller on a loss.
func BottleOfSuleimanEffect() Effect {
	return FlipCoin(You,
		OnWin(CreateToken("Djinn", 5, 5, []CardType{TypeArtifact, TypeCreature}, []string{"Djinn"}, Flying)),
		OnLoss(DealDamageToPlayers(Fixed(5), SelectController())),
	)
}

// HealingSalvePreventionEffect prevents the next 3 damage to a target this turn.
func HealingSalvePreventionEffect() Effect { return PreventDamageToTarget(Fixed(3)) }

// PandorasBoxEffect chooses a random creature card from all libraries, then
// gives each player who wins a coin flip a token copy of the chosen card.
func PandorasBoxEffect() Effect {
	return FuncEffect("choose a random creature card from all libraries; each player who wins a coin flip creates a token copy", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		var creatures []Card
		for _, player := range g.AllPlayers() {
			for _, card := range player.Library() {
				if card.HasType(TypeCreature) {
					creatures = append(creatures, card)
				}
			}
		}
		if len(creatures) == 0 {
			return nil
		}
		chosen := creatures[g.RandIntn(len(creatures))]
		for _, player := range g.AllPlayers() {
			if !g.FlipCoin(player.PlayerID()) {
				continue
			}
			copySource, err := CreateCard(chosen.Name())
			if err != nil {
				return err
			}
			token := NewToken(copySource.Name(), copySource.Power(), copySource.Toughness(), copySource.Types(), copySource.SubTypes())
			token.CloneFrom(copySource)
			token.SetOwner(player.PlayerID())
			g.PutOnBattlefieldBy(token, player.PlayerID(), player.PlayerID())
		}
		return nil
	})
}

// SindbadEffect draws a card and discards it if it isn't a land card.
func SindbadEffect() Effect {
	return FuncEffect("draw and reveal; discard if not land", EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
		player := g.GetPlayer(controller)
		if player == nil {
			return nil
		}
		card, ok := g.PlayerDrawCard(player)
		if ok && !card.HasType(TypeLand) {
			g.PlayerDiscardByEffect(player, card.ID(), sourceID)
		}
		return nil
	})
}

// TheHiveEffect creates a 1/1 flying Wasp artifact creature token.
func TheHiveEffect() Effect {
	return CreateToken("Wasp", 1, 1, []CardType{TypeArtifact, TypeCreature}, []string{"Insect"}, Flying)
}

// BerserkEffect doubles a target creature's power, grants trample, and destroys
// it at the beginning of the next end step if it attacked this turn.
func BerserkEffect() Effect {
	return CompositeEffects(
		"target creature gains trample and gets +X/+0; destroy it at end of turn if it attacked",
		GrantKeyword(Trample),
		DoubleTargetPower(),
		FuncEffect("destroy target creature at end of turn if it attacked", EffectProperties{}, func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			if len(targets) == 0 {
				return nil
			}
			targetID := targets[0]
			g.RegisterDelayedTrigger(&DelayedTrigger{
				EventType:  EvtEndStep,
				SourceID:   sourceID,
				Controller: controller,
				Effects: []Effect{FuncEffect("destroy it if it attacked this turn", EffectProperties{}, func(g2 *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
					if g2.HasAttackedThisTurn(targetID) {
						if permanent := g2.FindPermanent(targetID); permanent != nil {
							g2.DestroyPermanent(permanent)
						}
					}
					return nil
				})},
			})
			return nil
		}),
	)
}

// LaceEffect changes a target spell or permanent to the requested color.
func LaceEffect(color Color) Effect { return ChangeColorEffect(color) }

// LesserWerewolfCounterEffect puts a -0/-1 counter on a target creature.
func LesserWerewolfCounterEffect() Effect {
	return AddCounters(M0M1, Fixed(1)).Targeting(ToTarget())
}

// LightningBoltEffect deals 3 damage to any target.
func LightningBoltEffect() Effect { return DealDamage(Fixed(3)) }

// ProdigalSorcererEffect deals 1 damage to any target.
func ProdigalSorcererEffect() Effect { return DealDamage(Fixed(1)) }
