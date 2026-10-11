package mage

import (
	"fmt"
	"slices"
	"strings"
)

// PermanentChoiceRequest declares one required choice and its separate result.
type PermanentChoiceRequest struct {
	Player   PlayerRef
	Subjects PermanentSet
	Result   *PermanentChoiceBinding
}

// ChoosePermanentsAPNAP saves the choosers before choices in turn order.
// Later instructions can read each result. Choices do not move the subjects.
func ChoosePermanentsAPNAP(requests ...PermanentChoiceRequest) ResolvingEffect {
	return &apnapPermanentChoices{requests: slices.Clone(requests)}
}

type apnapPermanentChoices struct{ requests []PermanentChoiceRequest }

func (*apnapPermanentChoices) Text() string                 { return "choose permanents in turn order" }
func (*apnapPermanentChoices) Properties() EffectProperties { return EffectProperties{} }
func (e *apnapPermanentChoices) Apply(ctx *EffectContext) error {
	for _, request := range e.requests {
		if err := SnapshotPlayer(request.Player, request.Result.player).Apply(ctx); err != nil {
			return err
		}
	}
	for _, request := range e.requests {
		delete(ctx.Game.resolution.permanentBindings, request.Result.permanent)
	}
	previous := make([]string, 0, len(e.requests))
	for _, player := range []Player{ctx.Game.ActivePlayerObj(), ctx.Game.NonActivePlayerObj()} {
		if player == nil {
			continue
		}
		for _, request := range e.requests {
			chooser, ok := FromPlayer(request.Result.player).resolvePlayer(ctx)
			if !ok || chooser != player.PlayerID() {
				continue
			}
			reason := "Choose a permanent"
			if len(previous) > 0 {
				reason += ". Previous choices: " + strings.Join(previous, "; ")
			}
			choice := &permanentChoice{FromPlayer(request.Result.player), request.Subjects, request.Result.permanent}
			if err := choice.applyWithReason(ctx, reason); err != nil {
				return err
			}
			selection := "no permanent"
			if chosen, ok := PermanentChosenIn(request.Result).resolvePermanent(ctx); ok {
				selection = fmt.Sprintf("%s (%s)", chosen.Name(), chosen.ID())
			}
			previous = append(previous, player.Name()+": "+selection)
		}
	}
	return nil
}
func (e *apnapPermanentChoices) inspectDefinition() definitionNode {
	n := definitionNode{Role: "apnap-permanent-choices", Text: e.Text()}
	if len(e.requests) == 0 {
		n.problem = "supply permanent choices"
	}
	seen := make(map[*PermanentChoiceBinding]bool)
	for _, request := range e.requests {
		if request.Result == nil {
			n.problem = "supply non-nil choice results"
			continue
		}
		if seen[request.Result] {
			n.problem = "supply separate choice results"
		}
		seen[request.Result] = true
		n.children = append(n.children, definitionEdge{"choice", ChoosePermanentForPlayer(request.Player, request.Subjects, request.Result)})
		n.bindingWrites = append(n.bindingWrites, request.Result.player, request.Result.permanent)
	}
	return n
}
func (e *apnapPermanentChoices) refersToSourcePermanent() bool {
	for _, request := range e.requests {
		if playerRefReadsSourcePermanent(request.Player) || permanentSetReadsSourcePermanent(request.Subjects) {
			return true
		}
	}
	return false
}
