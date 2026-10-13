package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// ResolutionPaymentOption declares one complete resolution payment.
type ResolutionPaymentOption struct {
	costs        CostGroup
	action       ResolutionActionPayment
	actionOption bool
}

// CostPaymentOption keeps ordinary costs in the shared payment transaction.
func CostPaymentOption(costs CostGroup) ResolutionPaymentOption {
	return ResolutionPaymentOption{costs: costs}
}

// ActionPaymentOption keeps acceptance separate from the action outcome.
func ActionPaymentOption(action ResolutionActionPayment) ResolutionPaymentOption {
	return ResolutionPaymentOption{action: action, actionOption: true}
}

// PayAnyAlternativeOrElseWithPayer offers complete alternatives in turn order.
// The first payment saves its payer for paid. Otherwise, unpaid runs once.
func PayAnyAlternativeOrElseWithPayer(players PlayerSet, options []ResolutionPaymentOption, payer *PlayerBinding, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &eligiblePlayerPayment{players: players, alternatives: &resolutionPaymentAlternatives{options: slices.Clone(options)}, payer: payer, scopedPayer: true, paid: paid, unpaid: unpaid}
}

type resolutionPaymentAlternatives struct{ options []ResolutionPaymentOption }

func (e *resolutionPaymentAlternatives) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-payment-alternatives"}
	if len(e.options) == 0 {
		n.problem = "supply at least one payment option"
	}
	for i, option := range e.options {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("options[%d]", i), option})
	}
	return n
}
func (o ResolutionPaymentOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-payment-option"}
	if o.actionOption {
		n.children = []definitionEdge{{"action", o.action}}
	} else {
		n.children = []definitionEdge{{"costs", o.costs}}
	}
	return n
}
func (o ResolutionPaymentOption) properties() EffectProperties {
	if o.actionOption {
		return PayActionOrElse(You, o.action, nil, nil).Properties()
	}
	return EffectProperties{}
}
func (e *resolutionPaymentAlternatives) offer(ctx *EffectContext, eligible []uuid.UUID, payer *PlayerBinding) bool {
	return offerEligibleResolutionPayment(ctx, eligible, payer, func(id uuid.UUID) bool { return e.offerPlayer(ctx, id) })
}

func (e *resolutionPaymentAlternatives) offerPlayer(ctx *EffectContext, id uuid.UUID) bool {
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return false
	}
	options := make([]ResolutionPaymentOption, 0, len(e.options))
	actions := make([]preparedResolutionAction, 0, len(e.options))
	texts := make([]string, 0, len(e.options))
	for _, option := range e.options {
		prepared := preparedResolutionAction{}
		available := true
		text := ""
		if option.actionOption {
			prepared, available = option.action.prepareResolutionAction(ctx)
			if available {
				text = prepared.prompt(ctx.Game)
			}
		} else {
			parts := make([]string, len(option.costs.costs))
			for i, cost := range option.costs.costs {
				parts[i] = cost.Text()
				if !canPayWithManaPolicy(cost, ctx.SourceID, id, ctx.Game, false) {
					available = false
				}
			}
			text = "pay " + strings.Join(parts, ", ")
		}
		if available {
			options = append(options, option)
			actions = append(actions, prepared)
			texts = append(texts, text)
		}
	}
	if len(options) == 0 || !player.ChooseMayAbility(strings.Join(texts, " or ")) {
		return false
	}
	selected := 0
	if len(options) > 1 {
		selected = player.ChooseMode(texts, "choose a resolution payment")
	}
	if selected < 0 || selected >= len(options) {
		return false
	}
	option := options[selected]
	if !option.actionOption {
		return tryResolutionCostPayment(ctx, id, option.costs.costs, true)
	}
	if !actions[selected].available(ctx.Game) {
		return false
	}
	actions[selected].pay(ctx)
	return true
}
