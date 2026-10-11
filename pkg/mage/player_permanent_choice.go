package mage

import (
	"slices"

	"github.com/google/uuid"
)

// PermanentChoiceBinding saves a chooser and an exact permanent visit.
// Each resolution frame keeps separate values in the existing binding maps.
type PermanentChoiceBinding struct {
	player    *PlayerBinding
	permanent *PermanentBinding
}

// BindPermanentChoice declares one player and permanent choice result.
func BindPermanentChoice(name string) *PermanentChoiceBinding {
	return &PermanentChoiceBinding{BindPlayer(name + " chooser"), BindPermanent(name)}
}

// ChoosePermanentForPlayer saves the chooser before the required choice.
// An empty choice retains the chooser and clears the permanent result.
func ChoosePermanentForPlayer(player PlayerRef, subjects PermanentSet, result *PermanentChoiceBinding) ResolvingEffect {
	if result == nil {
		return Sequence(SnapshotPlayer(player, nil), ChoosePermanent(player, subjects, nil))
	}
	return Sequence(SnapshotPlayer(player, result.player), ChoosePermanent(FromPlayer(result.player), subjects, result.permanent))
}

// PermanentChosenIn reads the permanent from one choice result.
func PermanentChosenIn(result *PermanentChoiceBinding) PermanentRef {
	if result == nil {
		return From(nil)
	}
	return From(result.permanent)
}

type playerChosenPermanentRef struct {
	player  PlayerRef
	choices []*PermanentChoiceBinding
}

// PermanentChosenByPlayer reads the last preceding choice by that player.
// An empty latest choice cannot supply an earlier permanent.
func PermanentChosenByPlayer(player PlayerRef, choices ...*PermanentChoiceBinding) PermanentRef {
	return playerChosenPermanentRef{player, slices.Clone(choices)}
}
func (playerChosenPermanentRef) permanentRef() {}
func (r playerChosenPermanentRef) selected(ctx *EffectContext) *PermanentBinding {
	id, ok := r.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	for _, choice := range slices.Backward(r.choices) {
		if chooser, exists := ctx.Game.resolution.playerBindings[choice.player]; exists && chooser == id {
			return choice.permanent
		}
	}
	return nil
}
func (r playerChosenPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	return From(r.selected(ctx)).resolvePermanent(ctx)
}
func (r playerChosenPermanentRef) informationIdentity(ctx *EffectContext) (capturedPermanent, bool) {
	return savedPermanentRef{r.selected(ctx)}.informationIdentity(ctx)
}
func (r playerChosenPermanentRef) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-ref", Text: "permanent chosen by player", children: []definitionEdge{{"player", r.player}}}
	if len(r.choices) == 0 {
		n.problem = "supply preceding permanent choices"
	}
	for _, choice := range r.choices {
		if choice == nil {
			n.problem = "supply non-nil permanent choices"
			continue
		}
		n.children = append(n.children, definitionEdge{"chooser", FromPlayer(choice.player)}, definitionEdge{"choice", From(choice.permanent)})
	}
	return n
}

func (playerChosenPermanentRef) damageRecipient() {}
func (r playerChosenPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ID(), true
}
