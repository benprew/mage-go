package interactive

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

// ObserveLibraryLook shows the private cards before the next resolution choice.
func (p *HumanPlayer) ObserveLibraryLook(cards []mage.Card, _ mage.GameReader) {
	details := make([]string, len(cards))
	for i, c := range cards {
		details[i] = c.Name() + " " + c.ManaCost().String() + ": " + buildRulesText(c)
	}
	p.requestChoice(ChoiceRequest{Type: ChoiceMode, Reason: "Look at these cards:\n" + strings.Join(details, "\n"), Options: []ChoiceOption{{Label: "Continue"}}})
}

// ChooseLibraryTopOrder keeps all inspected cards on top of the library.
func (p *HumanPlayer) ChooseLibraryTopOrder(cards []mage.Card, g mage.GameReader) []uuid.UUID {
	remaining := slices.Clone(cards)
	order := make([]uuid.UUID, 0, len(cards))
	for len(remaining) > 1 {
		chosen := p.ChooseCardFromLibrary(remaining, "choose the next card from the top", g)
		index := max(slices.IndexFunc(remaining, func(c mage.Card) bool { return chosen != nil && c.ID() == chosen.ID() }), 0)
		order = append(order, remaining[index].ID())
		remaining = slices.Delete(remaining, index, index+1)
	}
	if len(remaining) == 1 {
		order = append(order, remaining[0].ID())
	}
	return order
}
