package promo

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

func init() {
	registerSpells()
}

func registerSpells() {

	// Sewers of Estark {2}{B}{B}
	// Instant
	// Choose target creature. If it's attacking, it can't be blocked this turn. If it's blocking, prevent all combat damage that would be dealt this combat by it and each creature it's blocking.
	Register("Sewers of Estark", func() Card {
		return NewInstant("Sewers of Estark", "{2}{B}{B}",
			NewTargetedSpell(TargetCreature(), FuncEffect(
				"if target creature is attacking, it can't be blocked this turn; if it's blocking, prevent combat damage by it and creatures it's blocking this combat",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, _ uuid.UUID, targets []uuid.UUID) error {
					if len(targets) == 0 {
						return nil
					}
					targetID := targets[0]
					if g.IsAttackingInCombat(targetID) {
						effect := TemporaryKeyword(targetID, UnblockableKW)
						effect.SetSourceID(sourceID)
						g.AddContinuousEffect(effect)
					}
					if !g.IsBlockingInCombat(targetID) {
						return nil
					}

					preventedSources := map[uuid.UUID]bool{targetID: true}
					for _, group := range g.CombatGroups() {
						if slices.Contains(group.BlockerIDs, targetID) {
							preventedSources[group.AttackerID] = true
						}
					}
					for preventedID := range preventedSources {
						id := preventedID
						effect := FuncContinuousEffect(LayerAbility, EndOfCombat, func(g *Game, _ uuid.UUID) error {
							g.AddDamagePreventionRule(
								WithCombatOnly(),
								WithFrom(NewPermanentFilter("Sewers of Estark prevented source", func(p *Permanent, _ *Game) bool {
									return p.ID() == id
								})),
							)
							return nil
						})
						effect.SetSourceID(sourceID)
						g.AddContinuousEffect(effect)
					}
					return nil
				},
			)),
		)
	})

}
