package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// RepeatHeadsOrTailsDamage repeats complete coin rounds until all coins are heads.
// Each round deals one simultaneous damage batch to the players with tails.
// These flips have no winner or loser. Results stay local to the round.
func RepeatHeadsOrTailsDamage(source DamageSourceRef, amount ResolutionValue, players ...PlayerRef) ResolvingEffect {
	return &headsOrTailsDamage{source: source, amount: amount, players: slices.Clone(players)}
}

type headsOrTailsDamage struct {
	source  DamageSourceRef
	amount  ResolutionValue
	players []PlayerRef
}

func (*headsOrTailsDamage) Text() string {
	return "flip coins and deal damage for tails until all coins are heads"
}
func (*headsOrTailsDamage) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *headsOrTailsDamage) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{
		{"source", damageSourceDefinition{e.source}}, {"amount", e.amount},
	}}
	if len(e.players) == 0 {
		n.problem = "supply at least one player reference"
	}
	for i, player := range e.players {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("players[%d]", i), player})
	}
	return n
}
func (e *headsOrTailsDamage) Apply(ctx *EffectContext) error {
	var participants []uuid.UUID
	for _, ref := range e.players {
		id, ok := ref.resolvePlayer(ctx)
		if ok && ctx.Game.GetPlayer(id) != nil && !slices.Contains(participants, id) {
			participants = append(participants, id)
		}
	}
	if len(participants) == 0 {
		return nil
	}
	for {
		var tails []uuid.UUID
		for _, id := range participants {
			if !ctx.Game.FlipCoin(id) {
				tails = append(tails, id)
			}
		}
		if len(tails) == 0 {
			return nil
		}
		amount := e.amount.resolveValue(ctx)
		source := e.source.resolveDamageSource(ctx)
		if amount > 0 && source.Valid() {
			assignments := make([]DamageAssignment, 0, len(tails))
			for _, id := range tails {
				assignments = append(assignments, PlayerDamageAssignment(ctx.Game.GetPlayer(id), amount, source))
			}
			ctx.Game.DealSimultaneousDamage(assignments...)
		}
	}
}
func (e *headsOrTailsDamage) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || resolutionValueReadsSourcePermanent(e.amount) || slices.ContainsFunc(e.players, playerRefReadsSourcePermanent)
}
