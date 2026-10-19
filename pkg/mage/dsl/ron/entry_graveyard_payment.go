package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileEntryGraveyardPayment(n Node) (optionFactory, error) {
	operations := n.Args[1].Args
	if len(operations) != 2 {
		return nil, problem(n, "entry exile payment requires a payment and its two branches")
	}
	payment, branch := operations[0], operations[1]
	if payment.Name != "MustCost" {
		return nil, unsupported(payment, "entry mandatory payment")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	exile := payment.Args[0]
	if exile.Name != "Exile" {
		return nil, unsupported(exile, "entry payment")
	}
	if err := arity(exile, 2); err != nil {
		return nil, err
	}
	objects, err := list(exile.Args[0])
	if err != nil {
		return nil, err
	}
	modifications, err := list(exile.Args[1])
	if err != nil {
		return nil, err
	}
	if len(objects) != 1 || len(modifications) != 0 {
		return nil, problem(exile, "entry payment requires one graveyard group without modifications")
	}
	object := objects[0]
	if object.Name != "NumberCardsInGraveyards" {
		return nil, unsupported(object, "entry payment subjects")
	}
	if err := arity(object, 2); err != nil {
		return nil, err
	}
	if object.Args[0].Kind != Symbol || object.Args[0].Name != "ValueX" {
		return nil, unsupported(object.Args[0], "entry payment amount")
	}
	filter := object.Args[1]
	if filter.Name != "And" {
		return nil, unsupported(filter, "entry payment filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "entry payment requires a card filter and your graveyard")
	}
	card, zone := parts[0], parts[1]
	if card.Name == "InAPlayersGraveyard" {
		card, zone = zone, card
	}
	if zone.Name != "InAPlayersGraveyard" {
		return nil, unsupported(zone, "entry payment graveyard")
	}
	if err := arity(zone, 1); err != nil {
		return nil, err
	}
	player := zone.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "entry payment player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
		return nil, unsupported(player.Args[0], "entry payment player")
	}
	subjects, err := compileHandCardFilter(card)
	if err != nil {
		return nil, err
	}
	if branch.Name != "IfElse" {
		return nil, unsupported(branch, "entry payment branches")
	}
	if err := arity(branch, 3); err != nil {
		return nil, err
	}
	if branch.Args[0].Kind != Symbol || branch.Args[0].Name != "CostWasPaid" {
		return nil, unsupported(branch.Args[0], "entry payment result")
	}
	paid, err := list(branch.Args[1])
	if err != nil {
		return nil, err
	}
	unpaid, err := list(branch.Args[2])
	if err != nil {
		return nil, err
	}
	if len(paid) != 1 || len(unpaid) != 1 || unpaid[0].Kind != Symbol || unpaid[0].Name != "PutIntoGraveyardInstead" {
		return nil, problem(branch, "entry payment requires counter allocation or failed graveyard entry")
	}
	allocation := paid[0]
	if allocation.Name != "EntersWithNumberCombinationCountersOfChoice" {
		return nil, unsupported(allocation, "entry payment continuation")
	}
	if err := arity(allocation, 2); err != nil {
		return nil, err
	}
	count := allocation.Args[0]
	if count.Name != "NumberOfCardsOfTypeExiledThisWay" {
		return nil, unsupported(count, "entry exile result")
	}
	if err := arity(count, 1); err != nil {
		return nil, err
	}
	if definitionKey(count.Args[0]) != definitionKey(card) {
		return nil, problem(count, "entry exile count must match the paid card filter")
	}
	choices, err := list(allocation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(choices) == 0 {
		return nil, problem(allocation, "supply counter choices")
	}
	var counters []core.CounterType
	for _, choice := range choices {
		counter, err := compilePermanentCounterType(choice)
		if err != nil {
			return nil, err
		}
		if slices.Contains(counters, counter) {
			return nil, problem(choice, "duplicate entry counter choice")
		}
		counters = append(counters, counter)
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.ExileGraveyardAsEntersWithCounterChoice(mage.AnnouncedXForEnteringPermanent(), subjects, counters...))
	}, nil
}
