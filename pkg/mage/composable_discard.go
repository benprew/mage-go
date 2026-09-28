package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type discardSubjectEffect struct {
	subject       PlayerRef
	amount        ResolutionValue
	cardResult    *CardBinding
	optionCount   int
	invalidOption bool
}

// DiscardCardResultOption selects a card result from one discard instruction.
type DiscardCardResultOption interface {
	discardCardResultOption()
	applyDiscardCardResult(*discardSubjectEffect)
}

type discardedCardOption struct{ binding *CardBinding }

// DiscardCardInto saves a card that the instruction discards into a graveyard.
func DiscardCardInto(binding *CardBinding) DiscardCardResultOption {
	return discardedCardOption{binding: binding}
}
func (discardedCardOption) discardCardResultOption() {}
func (o discardedCardOption) applyDiscardCardResult(e *discardSubjectEffect) {
	e.cardResult = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func (e *discardSubjectEffect) Text() string {
	return "discard for " + InspectDefinition(e.subject).Text
}
func (*discardSubjectEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardSubjectEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
	if e.invalidOption {
		n.problem = "supply a non-nil discard card binding"
	} else if e.optionCount > 1 {
		n.problem = "supply the discard card result only once"
	} else if e.cardResult != nil {
		if fixed, ok := e.amount.(fixedValue); !ok || fixed.n != 1 {
			n.problem = "discard one card when saving its card result"
		}
	}
	if e.cardResult != nil {
		n.bindingWrite = e.cardResult
		n.children = append(n.children, definitionEdge{"card result", e.cardResult})
	}
	return n
}
func (e *discardSubjectEffect) Apply(ctx *EffectContext) error {
	if e.cardResult != nil {
		delete(ctx.Game.resolution.cardBindings, e.cardResult)
	}
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	if e.cardResult != nil {
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			return nil
		}
		chosen := player.ChooseCardsFromHand(1, "discard", ctx.Game)
		for _, card := range chosen[:min(1, len(chosen))] {
			if discarded, ok := ctx.Game.PlayerDiscardByEffect(player, card.ID(), ctx.SourceID); ok {
				if visit := ctx.Game.graveyardVisit(discarded.ID()); visit != uuid.Nil {
					if ctx.Game.resolution.cardBindings == nil {
						ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
					}
					ctx.Game.resolution.cardBindings[e.cardResult] = capturedCard{id: discarded.ID(), visit: visit, zone: core.ZoneGraveyard}
				}
			}
		}
		return nil
	}
	child := *ctx
	child.Targets = []uuid.UUID{id}
	return DiscardCards(Fixed(amount)).Apply(&child)
}
