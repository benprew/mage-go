package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// LibrarySearchKind identifies a complete single-card search and shuffle.
type LibrarySearchKind uint8

const (
	SearchAnyCardToHand LibrarySearchKind = iota + 1
	SearchBasicLandToBattlefield
)

// LibrarySearchInstruction retains a search of the resolving controller's library.
// The destination precedes the shuffle. No search choice is stored here.
type LibrarySearchInstruction struct{ Kind LibrarySearchKind }

// Effect builds a native search. Unsupported kinds return nil.
func (instruction LibrarySearchInstruction) Effect() ResolvingEffect {
	switch instruction.Kind {
	case SearchAnyCardToHand, SearchBasicLandToBattlefield:
		return &librarySearchInstructionEffect{kind: instruction.Kind}
	default:
		return nil
	}
}

type librarySearchInstructionEffect struct{ kind LibrarySearchKind }

func (e *librarySearchInstructionEffect) execution() Effect {
	if e.kind == SearchAnyCardToHand {
		return SearchLibraryToHand()
	}
	return SearchLibraryToBattlefield(NewCardFilter("basic land", func(card Card) bool {
		return card.HasType(core.TypeLand) && card.HasSuperType(core.SuperBasic)
	}))
}

func (e *librarySearchInstructionEffect) Apply(ctx *EffectContext) error {
	return e.execution().Apply(ctx)
}
func (e *librarySearchInstructionEffect) Text() string { return e.execution().Text() }
func (e *librarySearchInstructionEffect) Properties() EffectProperties {
	return e.execution().Properties()
}

// InspectLibrarySearchInstruction reads a complete native search without choices.
// Opaque filters and other search procedures require separate contracts.
func InspectLibrarySearchInstruction(effect ResolvingEffect) (LibrarySearchInstruction, bool) {
	switch e := effect.(type) {
	case *librarySearchInstructionEffect:
		if e != nil && (e.kind == SearchAnyCardToHand || e.kind == SearchBasicLandToBattlefield) {
			return LibrarySearchInstruction{Kind: e.kind}, true
		}
	case *searchLibraryEffect:
		if e != nil {
			return LibrarySearchInstruction{Kind: SearchAnyCardToHand}, true
		}
	}
	return LibrarySearchInstruction{}, false
}
