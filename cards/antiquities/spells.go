package antiquities

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/dsl"
)

func init() {
	registerSpells()
}

func registerSpells() {
	// ===== INSTANTS =====

	// Artifact Blast {R}
	// Instant
	// Counter target artifact spell.
	Register("Artifact Blast", func() Card {
		return NewInstant("Artifact Blast", "{R}",
			NewTargetedSpell(TargetSpellOnStack(IsArtifactCard), CounterSpell()),
		)
	})

	// Crumble {G}
	// Instant
	// Destroy target artifact. It can't be regenerated. That artifact's controller gains life
	// equal to its mana value.
	Register("Crumble", func() Card {
		return NewInstant("Crumble", "{G}",
			NewTargetedSpell(TargetArtifact(), CrumbleEffect()),
		)
	})

	// Hurkyl's Recall {1}{U}
	// Instant
	// Return all artifacts target player owns to their hand.
	Register("Hurkyl's Recall", func() Card {
		return NewInstant("Hurkyl's Recall", "{1}{U}",
			// TODO: convert to pipeline — needs "bounce all matching owned by target player" step
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"return all artifacts target player owns to their hand",
				EffectProperties{Outcome: OutcomeDetriment},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayerID := targets[0]
					targetPlayer := g.GetPlayer(targetPlayerID)
					if targetPlayer == nil {
						return nil
					}
					var toReturn []*Permanent
					for _, perm := range g.FilterBattlefield(IsArtifact) {
						if perm.Card.Owner() == targetPlayerID {
							toReturn = append(toReturn, perm)
						}
					}
					for _, perm := range toReturn {
						g.RemoveFromBattlefield(perm)
						owner := g.GetPlayer(perm.Card.Owner())
						if owner != nil {
							owner.AddToHand(perm.Card)
						}
					}
					return nil
				})),
		)
	})

	// Reverse Polarity {W}{W}
	// Instant
	// You gain X life, where X is twice the damage dealt to you so far this turn by artifacts.
	Register("Reverse Polarity", func() Card {
		return NewInstant("Reverse Polarity", "{W}{W}",
			// TODO: convert to pipeline — needs "get artifact damage taken" ValueSource
			NewSpellAbility(FuncEffect("gain life equal to twice artifact damage",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					artDmg := g.GetArtifactDamageTaken(controller)
					if artDmg > 0 {
						p := g.GetPlayer(controller)
						if p != nil {
							g.PlayerGainLife(p, artDmg*2)
						}
					}
					return nil
				})),
		)
	})

	// ===== SORCERIES =====

	// Detonate {X}{R}
	// Sorcery
	// Destroy target artifact with mana value X. It can't be regenerated. Detonate deals X damage
	// to that artifact's controller.
	Register("Detonate", func() Card {
		return NewSorcery("Detonate", "{X}{R}",
			NewTargetedSpell(TargetArtifactWithManaValueX(),
				Pipeline("destroy target artifact with CMC X; deal X damage to controller",
					EffectProperties{Outcome: OutcomeDetriment},
					SnapshotPermanent(SelectTarget, "victim"),
					DestroyTargetNoRegenStep(),
					DealDamageToPlayersFromVar("victim.cmc", VarPlayer("victim.controller")),
				)),
		)
	})

	// Drafna's Restoration {U}
	// Sorcery
	// Put any number of target artifact cards from target player's graveyard on top of their
	// library in any order.
	// XXX: Oracle says "any number of target artifact cards from target player's graveyard" —
	// this requires dependent multi-targeting (second target pool depends on chosen player).
	// The engine doesn't support dependent targets. Currently uses resolution-time choices.
	// The order placed on top is the order chosen (last chosen = top).
	Register("Drafna's Restoration", func() Card {
		return NewSorcery("Drafna's Restoration", "{U}",
			// TODO: convert to pipeline — needs "choose cards from graveyard" iterative step
			NewTargetedSpell(TargetPlayer(), FuncEffect(
				"put artifact cards from graveyard on top of library",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetPlayer := g.GetPlayer(targets[0])
					if targetPlayer == nil {
						return nil
					}
					caster := g.GetPlayer(controller)
					if caster == nil {
						return nil
					}
					// Collect artifact cards from graveyard
					var artifactCards []Card
					for _, card := range targetPlayer.Graveyard() {
						if card.HasType(TypeArtifact) {
							artifactCards = append(artifactCards, card)
						}
					}
					if len(artifactCards) == 0 {
						return nil
					}
					// Caster chooses which artifact cards to put back, one at a time
					// (last chosen ends up on top of library)
					var chosen []Card
					for len(artifactCards) > 0 {
						if !caster.ChooseMayAbility("put an artifact card on top of library") {
							break
						}
						pick := caster.ChooseCardFromLibrary(artifactCards, "choose artifact card to put on top", g)
						if pick == nil {
							break
						}
						chosen = append(chosen, pick)
						// Remove from candidates
						var remaining []Card
						for _, c := range artifactCards {
							if c.ID() != pick.ID() {
								remaining = append(remaining, c)
							}
						}
						artifactCards = remaining
					}
					if len(chosen) == 0 {
						return nil
					}
					// Remove chosen cards from graveyard
					for _, card := range chosen {
						targetPlayer.RemoveFromGraveyard(card.ID())
					}
					// Put chosen cards on top of library in order (first chosen = bottom, last = top)
					lib := targetPlayer.Library()
					lib = append(lib, chosen...)
					targetPlayer.SetLibrary(lib)
					return nil
				})),
		)
	})

	// Reconstruction {U}
	// Sorcery
	// Return target artifact card from your graveyard to your hand.
	Register("Reconstruction", func() Card {
		return NewSorcery("Reconstruction", "{U}",
			NewTargetedSpell(TargetCardInYourGraveyard(IsArtifactCard), ReturnFromGraveyardToHandTarget()),
		)
	})

	// Shatterstorm {2}{R}{R}
	// Sorcery
	// Destroy all artifacts. They can't be regenerated.
	Register("Shatterstorm", func() Card {
		return NewSorcery("Shatterstorm", "{2}{R}{R}",
			NewSpellAbility(DestroyAllMatchingNoRegen(IsArtifact, "destroy all artifacts; they can't be regenerated")),
		)
	})

	// Transmute Artifact {U}{U}
	// Sorcery
	// Sacrifice an artifact. If you do, search your library for an artifact card. If that card's
	// mana value is less than or equal to the sacrificed artifact's mana value, put it onto the
	// battlefield. If not, you may pay {X}, where X is the difference. If you pay, put it onto
	// the battlefield. If you don't, put it into its owner's graveyard. Then shuffle.
	Register("Transmute Artifact", func() Card {
		return NewSorcery("Transmute Artifact", "{U}{U}",
			// TODO: convert to pipeline — needs "search library" + conditional payment steps
			NewSpellAbility(FuncEffect("search library for artifact, put on battlefield",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					p := g.GetPlayer(controller)
					if p == nil {
						return nil
					}
					sacrificedCMC := g.XValue()
					lib := p.Library()
					var candidates []Card
					for _, c := range lib {
						if c.HasType(TypeArtifact) {
							candidates = append(candidates, c)
						}
					}
					if len(candidates) == 0 {
						p.ShuffleLibrary()
						return nil
					}
					chosen := p.ChooseCardFromLibrary(candidates, "search for artifact", g)
					if chosen == nil {
						p.ShuffleLibrary()
						return nil
					}
					// Remove from library
					newLib := make([]Card, 0, len(lib)-1)
					for _, c := range lib {
						if c.ID() != chosen.ID() {
							newLib = append(newLib, c)
						}
					}
					p.SetLibrary(newLib)
					p.ShuffleLibrary()
					chosenCMC := chosen.ManaCost().CMC()
					if chosenCMC <= sacrificedCMC {
						// Free — put directly onto battlefield
						g.PutOnBattlefield(chosen, controller)
					} else {
						// Must pay the difference or put into graveyard
						difference := chosenCMC - sacrificedCMC
						if p.ManaPool().TotalMana() >= difference && p.ChooseMayAbility("pay the difference") {
							p.ManaPool().DrainGeneric(difference)
							g.PutOnBattlefield(chosen, controller)
						} else {
							p.AddToGraveyard(chosen)
						}
					}
					return nil
				})),
			WithAdditionalCost(&sacrificeArtifactCaptureCMCCost{}),
		)
	})
}
