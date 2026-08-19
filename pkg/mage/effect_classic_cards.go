package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// AladdinsRingEffect deals 4 damage to any target.
func AladdinsRingEffect() Effect { return DealDamage(Fixed(4)) }

// AncestralRecallEffect has a target player draw three cards.
func AncestralRecallEffect() Effect { return DrawCards(Fixed(3)) }

// BoomerangEffect returns a target permanent to its owner's hand.
func BoomerangEffect() Effect { return ReturnToHandTarget() }

// BottleOfSuleimanEffect flips a coin, creating a 5/5 flying Djinn artifact
// creature token on a win or dealing 5 damage to the controller on a loss.
func BottleOfSuleimanEffect() Effect {
	return IfElse(
		"flip coin: 5/5 Djinn or 5 damage",
		FlipCoinCond{},
		CreateToken("Djinn", 5, 5, []CardType{TypeArtifact, TypeCreature}, []string{"Djinn"}, Flying),
		DealDamageToPlayers(Fixed(5), SelectController()),
	)
}

// CrumbleEffect destroys a target artifact without regeneration and gives its
// controller life equal to its mana value.
func CrumbleEffect() Effect {
	return Pipeline(
		"destroy target artifact; its controller gains life equal to its mana value",
		EffectProperties{Outcome: OutcomeDetriment},
		SnapshotPermanent(SelectTarget, "target"),
		DestroyGatheredNoRegen("target"),
		GainLifeFromVar("target.controller", "target.cmc"),
	)
}

// DisenchantEffect destroys a target artifact or enchantment.
func DisenchantEffect() Effect { return DestroyTargetPermanent() }

// DisruptingScepterEffect has a target player discard a card.
func DisruptingScepterEffect() Effect { return DiscardCards(Fixed(1)) }

// FissureEffect destroys a target creature or land without regeneration.
func FissureEffect() Effect { return DestroyTargetNoRegen() }

// FogEffect prevents all combat damage that would be dealt this turn.
func FogEffect() Effect { return PreventAllCombatDamage() }

// HealingSalveGainEffect has a target player gain 3 life.
func HealingSalveGainEffect() Effect { return GainLifeTarget(Fixed(3)) }

// HealingSalvePreventionEffect prevents the next 3 damage to a target this turn.
func HealingSalvePreventionEffect() Effect { return PreventDamageToTarget(Fixed(3)) }

// MillstoneEffect has a target player mill two cards.
func MillstoneEffect() Effect { return MillTargetPlayer(Fixed(2)) }

// NevinyrralsDiskEffect destroys all artifacts, creatures, and enchantments.
func NevinyrralsDiskEffect() Effect {
	return CompositeEffects(
		"destroy all artifacts, creatures, and enchantments",
		DestroyAllCreatures(),
		DestroyAllEnchantments(),
		DestroyAllMatching(IsArtifact, "destroy all artifacts"),
	)
}

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
			g.PutOnBattlefield(token, player.PlayerID())
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

// BloodLustEffect gives a target creature +4/-4 if its toughness is at least
// five, or +4/-X where X leaves it with one toughness otherwise.
func BloodLustEffect() Effect {
	return FuncEffect("target creature gets +4/-4 or +4/-(toughness-1)", EffectProperties{Outcome: OutcomeDetriment}, func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
		if len(targets) == 0 {
			return nil
		}
		permanent := g.FindPermanent(targets[0])
		if permanent == nil {
			return nil
		}
		toughnessChange := -4
		if toughness := permanent.CurrentToughness(g); toughness < 5 {
			toughnessChange = -(toughness - 1)
		}
		boost := TemporaryBoost(permanent.ID(), 4, toughnessChange)
		boost.SetSourceID(sourceID)
		g.AddContinuousEffect(boost)
		g.ApplyContinuousEffects()
		return nil
	})
}

// FlyingCarpetEffect grants flying to a target creature until end of turn.
func FlyingCarpetEffect() Effect { return GrantKeyword(Flying) }

// GiantGrowthEffect gives a target creature +3/+3 until end of turn.
func GiantGrowthEffect() Effect { return Boost(Fixed(3), Fixed(3)) }

// HelmOfChatzukEffect grants banding to a target creature until end of turn.
func HelmOfChatzukEffect() Effect { return GrantKeyword(Banding) }

// HurrJackalEffect prevents a target creature from regenerating this turn.
func HurrJackalEffect() Effect { return GrantKeyword(CantRegenerate) }

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

// SorceressQueenEffect sets a target creature's base power and toughness to 0/2
// until end of turn.
func SorceressQueenEffect() Effect { return SetPTUntilEndOfTurn(0, 2, SelectTarget) }

// StaffOfZegonEffect gives a target creature -2/-0 until end of turn.
func StaffOfZegonEffect() Effect { return Boost(Fixed(-2), Fixed(0)) }

// SwordsToPlowsharesEffect exiles a target creature and gives its controller
// life equal to its power as it last existed on the battlefield.
func SwordsToPlowsharesEffect() Effect {
	return Pipeline(
		"exile target creature. Its controller gains life equal to its power",
		EffectProperties{Outcome: OutcomeDetriment, TargetPurposeOverride: AITargetExile},
		SnapshotPermanent(SelectTarget, "victim"),
		ExileGathered("victim"),
		GainLifeFromVar("victim.controller", "victim.power"),
	)
}

// TawnosWandEffect makes a target creature unblockable until end of turn.
func TawnosWandEffect() Effect { return MakeUnblockableUntilEndOfTurn() }

// TwiddleEffect taps or untaps a target permanent.
func TwiddleEffect() Effect { return TapOrUntapTarget() }

// UnsummonEffect returns a target creature to its owner's hand.
func UnsummonEffect() Effect { return ReturnToHandTarget() }
