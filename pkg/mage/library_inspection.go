package mage

import (
	"slices"

	"github.com/google/uuid"
)

// LibraryLookObserver receives private card copies during resolution.
type LibraryLookObserver interface{ ObserveLibraryLook([]Card, GameReader) }

// LibraryTopOrderChooser chooses a complete order without a bottom placement.
type LibraryTopOrderChooser interface {
	ChooseLibraryTopOrder([]Card, GameReader) []uuid.UUID
}

// LibraryLooks returns copies of the cards that the viewer saw during resolution.
// This history does not permit inspection of the current library order.
func (g *Game) LibraryLooks(viewer uuid.UUID) []PublicCardReveal {
	return clonePublicReveals(g.libraryLooks[viewer])
}

// LookAtLibraryTop shows at most count cards to the viewer. The viewer can
// change their order when reorder is true. The cards stay in the library.
func LookAtLibraryTop(viewer, library PlayerRef, count int, reorder bool) ResolvingEffect {
	return &libraryInspectionEffect{viewer, library, count, reorder}
}

type libraryInspectionEffect struct {
	viewer, library PlayerRef
	count           int
	reorder         bool
}

func (*libraryInspectionEffect) Text() string                 { return "look at the top cards of the selected library" }
func (*libraryInspectionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *libraryInspectionEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"viewer", e.viewer}, {"library", e.library}}}
	if e.count < 0 {
		n.problem = "library inspection count must not be negative"
	}
	return n
}
func (e *libraryInspectionEffect) Apply(ctx *EffectContext) error {
	viewer, owner := handOperationPlayer(e.viewer, ctx), handOperationPlayer(e.library, ctx)
	if viewer == nil || owner == nil || e.count <= 0 {
		return nil
	}
	cards := ctx.Game.RevealTopN(owner, e.count)
	if len(cards) == 0 {
		return nil
	}
	if ctx.Game.libraryLooks == nil {
		ctx.Game.libraryLooks = make(map[uuid.UUID][]PublicCardReveal)
	}
	for _, card := range cards {
		copied := card.Copy()
		copied.SetID(card.ID())
		ctx.Game.libraryLooks[viewer.PlayerID()] = append(ctx.Game.libraryLooks[viewer.PlayerID()], PublicCardReveal{CardID: card.ID(), Owner: card.Owner(), Name: card.Name(), Types: slices.Clone(card.Types()), Card: copied})
	}
	if observer, ok := viewer.(LibraryLookObserver); ok {
		snapshots := make([]Card, len(cards))
		for i, card := range cards {
			snapshots[i] = card.Copy()
			snapshots[i].SetID(card.ID())
		}
		observer.ObserveLibraryLook(snapshots, ctx.Game)
	}
	if e.reorder {
		var order []uuid.UUID
		if chooser, ok := viewer.(LibraryTopOrderChooser); ok {
			order = chooser.ChooseLibraryTopOrder(slices.Clone(cards), ctx.Game)
		} else {
			bottom, top := viewer.ChooseScryPlacement(slices.Clone(cards), "put on top in any order", ctx.Game)
			order = append(slices.Clone(bottom), top...)
		}
		byID := make(map[uuid.UUID]Card, len(cards))
		for _, card := range cards {
			byID[card.ID()] = card
		}
		ordered := make([]Card, 0, len(cards))
		valid := len(order) == len(cards)
		for _, id := range order {
			card, ok := byID[id]
			if !ok {
				valid = false
				break
			}
			ordered = append(ordered, card)
			delete(byID, id)
		}
		if valid {
			library := slices.Clone(owner.Library())
			copy(library, ordered)
			owner.SetLibrary(library)
		}
	}
	return nil
}

// MayShuffleLibrary lets the chooser decide if the selected player shuffles.
func MayShuffleLibrary(chooser, library PlayerRef) ResolvingEffect {
	return &mayShuffleLibraryEffect{chooser, library}
}

type mayShuffleLibraryEffect struct{ chooser, library PlayerRef }

func (*mayShuffleLibraryEffect) Text() string {
	return "you may have the selected player shuffle their library"
}
func (*mayShuffleLibraryEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *mayShuffleLibraryEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"chooser", e.chooser}, {"library", e.library}}}
}
func (e *mayShuffleLibraryEffect) Apply(ctx *EffectContext) error {
	chooser, owner := handOperationPlayer(e.chooser, ctx), handOperationPlayer(e.library, ctx)
	if chooser != nil && owner != nil && chooser.ChooseMayAbility("have "+owner.Name()+" shuffle their library") {
		owner.ShuffleLibrary()
	}
	return nil
}
