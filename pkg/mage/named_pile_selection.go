package mage

import (
	"fmt"
	"slices"
	"strings"
)

// ChooseNamedPermanentPile saves the visits in one required pile choice.
// The subject supplies the public choice label. Pile members do not move.
func ChooseNamedPermanentPile(player PlayerRef, subject PermanentRef, result *PermanentSetBinding, piles ...NamedPermanentPile) ResolvingEffect {
	return &namedPileSelection{player, subject, result, slices.Clone(piles)}
}

type namedPileSelection struct {
	player  PlayerRef
	subject PermanentRef
	result  *PermanentSetBinding
	piles   []NamedPermanentPile
}

func (*namedPileSelection) Text() string                 { return "choose a named permanent pile" }
func (*namedPileSelection) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *namedPileSelection) inspectDefinition() definitionNode {
	n := definitionNode{Role: "named-pile-selection", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"subject", e.subject}, {"result", e.result}}}
	if len(e.piles) < 2 {
		n.problem = "supply at least two named piles"
	}
	names := make(map[string]bool, len(e.piles))
	results := make(map[*PermanentSetBinding]bool, len(e.piles))
	for _, pile := range e.piles {
		if strings.TrimSpace(pile.Name) == "" || names[pile.Name] {
			n.problem = "supply distinct nonempty pile names"
		}
		if pile.Result == nil || results[pile.Result] || pile.Result == e.result {
			n.problem = "supply distinct non-nil pile inputs and a separate result"
		}
		names[pile.Name] = true
		results[pile.Result] = true
		n.children = append(n.children, definitionEdge{"pile input", FromPermanentSet(pile.Result)})
	}
	return n
}
func (e *namedPileSelection) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentRefReadsSourcePermanent(e.subject)
}
func (e *namedPileSelection) Apply(ctx *EffectContext) error {
	saved := make([][]capturedPermanent, len(e.piles))
	names := make([]string, len(e.piles))
	for i, pile := range e.piles {
		saved[i] = slices.Clone(ctx.Game.resolution.createdTokenBindings[pile.Result])
		names[i] = pile.Name
	}
	if ctx.Game.resolution.createdTokenBindings == nil {
		ctx.Game.resolution.createdTokenBindings = make(map[*PermanentSetBinding][]capturedPermanent)
	}
	delete(ctx.Game.resolution.createdTokenBindings, e.result)
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok || subject.PhasedOut {
		return nil
	}
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	descriptions := make([]string, len(e.piles))
	for i, pile := range e.piles {
		members := make([]string, 0, len(saved[i]))
		for _, visit := range saved[i] {
			live := ctx.Game.FindPermanent(visit.id)
			if live != nil && !live.PhasedOut && live.incarnationID == visit.incarnation {
				members = append(members, fmt.Sprintf("%s (%s)", live.Name(), live.ID()))
			}
		}
		contents := "empty"
		if len(members) > 0 {
			contents = strings.Join(members, ", ")
		}
		descriptions[i] = pile.Name + ": " + contents
	}
	reason := fmt.Sprintf("Choose a pile for %s (%s). Piles: %s", subject.Name(), subject.ID(), strings.Join(descriptions, "; "))
	choice := player.ChooseMode(names, reason)
	if choice < 0 || choice >= len(saved) {
		choice = 0
	}
	ctx.Game.resolution.createdTokenBindings[e.result] = saved[choice]
	return nil
}
