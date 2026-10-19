package mage

import (
	"errors"
	"slices"

	"github.com/google/uuid"
)

// SelectLibraryTopAndBottomRestRandomly keeps one inspected card on top.
// The remaining inspected cards go to the bottom in a random order.
func SelectLibraryTopAndBottomRestRandomly(viewer, library PlayerRef, count ResolutionValue) ResolvingEffect {
	return &libraryTopSelectionEffect{viewer, library, count}
}

type libraryTopSelectionEffect struct {
	viewer, library PlayerRef
	count           ResolutionValue
}

func (*libraryTopSelectionEffect) Text() string {
	return "choose one top card and put the rest on the bottom in a random order"
}
func (*libraryTopSelectionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *libraryTopSelectionEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"viewer", e.viewer}, {"library", e.library}, {"count", e.count}}}
	if count, ok := e.count.(fixedValue); ok && count.n < 0 {
		n.problem = "library selection count must not be negative"
	}
	return n
}
func (e *libraryTopSelectionEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.viewer) || playerRefReadsSourcePermanent(e.library) || resolutionValueReadsSourcePermanent(e.count)
}
func (e *libraryTopSelectionEffect) Apply(ctx *EffectContext) error {
	viewer, owner := handOperationPlayer(e.viewer, ctx), handOperationPlayer(e.library, ctx)
	if viewer == nil || owner == nil {
		return nil
	}
	count := e.count.resolveValue(ctx)
	cards := ctx.Game.RevealTopN(owner, count)
	if len(cards) == 0 {
		return nil
	}
	visitor, ok := owner.(interface{ libraryVisit(uuid.UUID) uuid.UUID })
	if !ok {
		return errors.New("library selection requires engine library-visit identity")
	}
	visits := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		visits[i] = visitor.libraryVisit(c.ID())
	}
	if err := LookAtLibraryTop(e.viewer, e.library, len(cards), false).Apply(ctx); err != nil {
		return err
	}
	snapshots := make([]Card, len(cards))
	for i, c := range cards {
		snapshots[i] = c.Copy()
		snapshots[i].SetID(c.ID())
	}
	chosen := viewer.ChooseCardFromLibrary(snapshots, "keep one card on top; put the rest on the bottom in a random order", ctx.Game)
	index := 0
	if chosen != nil {
		if found := slices.IndexFunc(cards, func(c Card) bool { return c.ID() == chosen.ID() }); found >= 0 {
			index = found
		}
	}
	current := ctx.Game.RevealTopN(owner, len(cards))
	if len(current) != len(cards) {
		return nil
	}
	for i, c := range cards {
		if current[i].ID() != c.ID() || visits[i] == uuid.Nil || visitor.libraryVisit(c.ID()) != visits[i] {
			return nil
		}
	}
	kept := make([]Card, 1, 1+len(owner.Library())-len(cards))
	kept[0] = cards[index]
	kept = append(kept, owner.Library()[len(cards):]...)
	owner.SetLibrary(kept)
	rest := make([]Card, 0, len(cards)-1)
	for i, c := range cards {
		if i != index {
			rest = append(rest, c)
		}
	}
	ctx.Game.PutOnBottomInRandomOrder(owner, rest)
	return nil
}
