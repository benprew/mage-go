package ai

import (
	"slices"
	"sort"
	"strconv"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

// RespondToChoice uses the AI choice methods for an interactive player.
// The controller shares the player's state so each choice uses the current hand.
func RespondToChoice(player *mage.BasePlayer, g *mage.Game, req interactive.ChoiceRequest) interactive.ChoiceResponse {
	response := interactive.ChoiceResponse{}
	if player == nil {
		return response
	}
	controller := &AIPlayer{BasePlayer: player}
	if req.Type == interactive.ChoiceMay {
		response.Accepted = controller.ChooseMayAbility(req.Reason)
		return response
	}
	if len(req.Options) == 0 {
		return response
	}
	switch req.Type {
	case interactive.ChoiceManaColor:
		color := controller.ChooseManaColor(req.Reason)
		response.SelectedColor = req.Options[0].Color
		for _, opt := range req.Options {
			if opt.Color == color {
				response.SelectedColor = color
				break
			}
		}
	case interactive.ChoiceMode:
		labels := make([]string, len(req.Options))
		for i, opt := range req.Options {
			labels[i] = opt.Label
		}
		response.SelectedIndex = controller.ChooseMode(labels, req.Reason)
	case interactive.ChoiceNumber:
		minimum, errMin := strconv.Atoi(req.Options[0].Label)
		maximum, errMax := strconv.Atoi(req.Options[len(req.Options)-1].Label)
		if errMin == nil && errMax == nil {
			number := controller.ChooseNumber(minimum, maximum, req.Reason)
			for i, opt := range req.Options {
				if opt.Label == strconv.Itoa(number) {
					response.SelectedIndex = i
					break
				}
			}
		}
	case interactive.ChoiceCardsFromHand, interactive.ChoiceCardFromLibrary:
		var cards []*mage.Card
		for _, opt := range req.Options {
			if g != nil {
				if card := g.FindCardAnywhere(opt.ID); card != nil {
					cards = append(cards, card)
				}
			}
		}
		amount := 1
		if req.Type == interactive.ChoiceCardsFromHand {
			amount = max(req.Amount, 0)
		}
		for range min(amount, len(cards)) {
			var card *mage.Card
			if req.Type == interactive.ChoiceCardsFromHand {
				card = controller.ChooseCardFromHand(cards, req.Reason, g)
			} else {
				card = controller.ChooseCardFromLibrary(cards, req.Reason, g)
			}
			if card == nil {
				break
			}
			response.SelectedIDs = append(response.SelectedIDs, card.ID())
			cards = slices.DeleteFunc(cards, func(c *mage.Card) bool { return c.ID() == card.ID() })
		}
	case interactive.ChoicePermanent:
		var ids []uuid.UUID
		var permanents []*mage.Permanent
		for _, opt := range req.Options {
			if opt.ID == uuid.Nil() {
				continue
			}
			ids = append(ids, opt.ID)
			if g != nil {
				if permanent := g.FindPermanent(opt.ID); permanent != nil {
					permanents = append(permanents, permanent)
				}
			}
		}
		if req.Reason == "Choose target" || len(permanents) != len(ids) {
			response.SelectedIDs = controller.ChooseTargets(ids, 1, 1, g)
		} else if permanent := controller.ChoosePermanent(permanents, req.Reason, g); permanent != nil {
			response.SelectedIDs = []uuid.UUID{permanent.ID()}
		}
	}
	return response
}

// SuggestedCombatDamage assigns damage in blocker order and includes zero amounts
// so an interface can show controls for each blocker.
func SuggestedCombatDamage(blockers []interactive.PermanentState, total int) map[uuid.UUID]int {
	assignment := make(map[uuid.UUID]int, len(blockers))
	remaining := max(total, 0)
	for i, blocker := range blockers {
		amount := min(max(blocker.Toughness-blocker.Damage, 0), remaining)
		if i == len(blockers)-1 {
			amount = remaining
		}
		assignment[blocker.ID] = amount
		remaining -= amount
	}
	return assignment
}

// CombatDamageOrder puts blockers with lethal damage before other blockers.
func CombatDamageOrder(blockers []interactive.PermanentState, assignment map[uuid.UUID]int) []uuid.UUID {
	blockers = slices.Clone(blockers)
	sort.SliceStable(blockers, func(i, j int) bool {
		iAssigned, jAssigned := assignment[blockers[i].ID], assignment[blockers[j].ID]
		iLethal := iAssigned >= max(blockers[i].Toughness-blockers[i].Damage, 0)
		jLethal := jAssigned >= max(blockers[j].Toughness-blockers[j].Damage, 0)
		if iLethal != jLethal {
			return iLethal
		}
		return iAssigned > jAssigned
	})
	order := make([]uuid.UUID, len(blockers))
	for i, blocker := range blockers {
		order[i] = blocker.ID
	}
	return order
}
