package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ExileGraveyardAsEntersWithCounterChoice requires payment before entry.
// Only completed exile moves supply counters. Failed payment prevents entry.
func ExileGraveyardAsEntersWithCounterChoice(amount EntryCounterValue, filter CardFilter, counters ...core.CounterType) Ability {
	return &entryGraveyardPaymentAbility{BaseAbility: BaseAbility{id: uuid.New(), abilityType: core.AbilityStatic}, amount: amount, filter: filter, counters: slices.Clone(counters)}
}

type entryGraveyardPaymentAbility struct {
	BaseAbility
	amount   EntryCounterValue
	filter   CardFilter
	counters []core.CounterType
}

func (a *entryGraveyardPaymentAbility) inspectDefinition() definitionNode {
	n := definitionNode{Role: "entry-graveyard-payment", Text: "exile graveyard cards and choose entry counters", battlefieldSource: true, children: []definitionEdge{{"amount", a.amount}}}
	if a.filter.IsZero() || len(a.counters) == 0 {
		n.problem = "supply a card filter and counter choices"
	}
	seen := map[core.CounterType]bool{}
	for _, counter := range a.counters {
		if counter < 0 || counter >= core.NumCounters || seen[counter] {
			n.problem = "supply distinct valid counter choices"
		}
		seen[counter] = true
	}
	return n
}

type entryGraveyardPaymentReplacement struct {
	replacementBase
	ability *entryGraveyardPaymentAbility
}

func (*entryGraveyardPaymentReplacement) IsActive(GameReader) bool { return true }
func (*entryGraveyardPaymentReplacement) Text() string             { return "exile graveyard cards as enters" }
func (*entryGraveyardPaymentReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*battlefieldEntryAction)
	return ok
}
func (r *entryGraveyardPaymentReplacement) Clone() ReplacementEffect { c := *r; return &c }
func (r *entryGraveyardPaymentReplacement) Replace(action Action, g *Game) Action {
	entry := action.(*battlefieldEntryAction)
	p := entry.permanent
	a := r.ability
	amount := max(0, a.amount.entryCounterAmount(entry.context))
	cost := ExileMatchingCardsFromSingleGraveyardCost(amount, a.filter, OnePlayer(You)).(*exileFromGraveyardCost)
	if err := cost.chooseSingleGraveyardWithFallback(p.ID(), p.ControllerID(), g, true); err != nil {
		return nil
	}
	if !cost.CanPay(p.ID(), p.ControllerID(), g) {
		return nil
	}
	count := g.exileGraveyardPaymentVisits(cost.selected, p.ID())
	player := g.GetPlayer(p.ControllerID())
	labels := make([]string, len(a.counters))
	for i, counter := range a.counters {
		labels[i] = counter.String()
	}
	for range count {
		choice := player.ChooseMode(labels, "Choose an entry counter")
		if choice < 0 || choice >= len(a.counters) {
			choice = 0
		}
		entry.context.entry.counterAmounts[a.counters[choice]]++
	}
	return action
}

// All payment visits are checked before replacements can change their moves.
func (g *Game) exileGraveyardPaymentVisits(selected []capturedCard, source uuid.UUID) int {
	proposals := make([]*CardZoneMoveAction, 0, len(selected))
	for _, subject := range selected {
		card := g.FindCardAnywhere(subject.id)
		proposal := &CardZoneMoveAction{source: source, card: subject.id, visit: subject.visit, owner: card.Owner(), from: core.ZoneGraveyard, destination: core.ZoneExile, exileSource: source}
		proposals = append(proposals, proposal)
	}
	var moves []*CardZoneMoveAction
	for _, proposal := range proposals {
		if move, ok := g.effects.ApplyReplacements(proposal, g).(*CardZoneMoveAction); ok && move != nil {
			moves = append(moves, move)
		}
	}
	count := 0
	g.withZoneChangeBatch(func() {
		var events []core.GameEvent
		counts := map[uuid.UUID]int{}
		for _, move := range moves {
			batch := g.performCardZoneMoveEvents(move)
			for _, event := range batch {
				if event.Type == core.EvtZoneChange && event.ToZone == core.ZoneExile {
					count++
				}
			}
			for _, event := range batch {
				if event.Type == core.EvtCardsLeftGraveyard {
					counts[event.PlayerID] += event.Amount
				} else {
					events = append(events, event)
				}
			}
		}
		for _, event := range events {
			g.FireEvent(event)
		}
		for _, player := range g.players {
			if amount := counts[player.PlayerID()]; amount > 0 {
				g.FireEvent(core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: amount})
			}
		}
	})
	return count
}
