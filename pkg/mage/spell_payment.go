package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// spellPaymentTransaction is the locked CR 601.2f total cost for one spell.
// prepareSpellPaymentTransaction resolves branching costs, combines every mana
// component, and proves the complete payment on a game clone before Commit
// mutates live state.
type spellPaymentTransaction struct {
	game       *Game
	card       Card
	controller uuid.UUID
	zone       Zone
	payment    *costPaymentTransaction
}

type spellPaymentSpec struct {
	Card           Card
	Controller     uuid.UUID
	Zone           Zone
	Mana           ManaCost
	LifeLoss       int
	Costs          []Cost
	Targets        []uuid.UUID
	XValue         int
	AnyColor       bool
	ChannelGeneric bool
}

func (g *Game) prepareSpellPaymentTransaction(spec spellPaymentSpec) (*spellPaymentTransaction, error) {
	if spec.Card == nil {
		return nil, ErrSourceNotFound
	}
	player := g.GetPlayer(spec.Controller)
	if player == nil {
		return nil, ErrPlayerNotFound
	}
	if extraMana, direct := directSpellManaCosts(spec.Costs, spec.XValue); direct {
		totalMana := resolvedSpellManaCost(spec.Mana, spec.XValue)
		addActionManaCost(&totalMana, extraMana)
		totalMana, lifeLoss := g.modifySpellTotalCost(spec, totalMana)
		spellCtx := SpellContextForCard(spec.Card)
		payment, err := g.prepareCostPaymentTransaction(costPaymentSpec{
			Controller:             spec.Controller,
			SourceID:               spec.Card.ID(),
			Mana:                   totalMana,
			LifeLoss:               lifeLoss,
			Hint:                   AutoTapHint{CastingCard: spec.Card.ID()},
			SpellContext:           spellCtx,
			ResetDrainedBeforeMana: true,
		}, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot pay total mana cost %s for %s: %w", totalMana, spec.Card.Name(), err)
		}
		return &spellPaymentTransaction{
			game:       g,
			card:       spec.Card,
			controller: spec.Controller,
			zone:       spec.Zone,
			payment:    payment,
		}, nil
	}

	validation := cloneGameForCostPayment(g)
	validation.resolution.SetX(spec.XValue)
	if validation.removeCardFromZone(spec.Controller, spec.Card.ID(), spec.Zone) == nil {
		return nil, fmt.Errorf("could not move %s from %s while validating its total cost", spec.Card.Name(), spec.Zone)
	}

	lockedCosts, outcomes, err := lockPaymentCosts(spec.Costs, spec.Card.ID(), spec.Controller, spec.Targets, spec.XValue, validation, player)
	if err != nil {
		return nil, err
	}
	totalMana := resolvedSpellManaCost(spec.Mana, spec.XValue)
	var nonMana []Cost
	for _, cost := range lockedCosts {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			addActionManaCost(&totalMana, resolvedSpellManaCost(payment.MC, spec.XValue))
		case *xManaCost:
			totalMana.Generic += spec.XValue
		default:
			nonMana = append(nonMana, cost)
		}
	}
	totalMana, lifeLoss := g.modifySpellTotalCost(spec, totalMana)

	spellCtx := SpellContextForCard(spec.Card)
	payment, err := g.prepareCostPaymentTransaction(costPaymentSpec{
		Controller:             spec.Controller,
		SourceID:               spec.Card.ID(),
		Mana:                   totalMana,
		LifeLoss:               lifeLoss,
		Costs:                  nonMana,
		OptionalOutcomes:       outcomes,
		Hint:                   AutoTapHint{CastingCard: spec.Card.ID()},
		SpellContext:           spellCtx,
		ResetDrainedBeforeMana: true,
	}, validation)
	if err != nil {
		return nil, fmt.Errorf("cannot pay total mana cost %s for %s: %w", totalMana, spec.Card.Name(), err)
	}

	return &spellPaymentTransaction{
		game:       g,
		card:       spec.Card,
		controller: spec.Controller,
		zone:       spec.Zone,
		payment:    payment,
	}, nil
}

// Cost changes use the complete mana total so reductions can cover added mana.
func (g *Game) modifySpellTotalCost(spec spellPaymentSpec, total ManaCost) (modified ManaCost, lifeLoss int) {
	for _, color := range cardColors(spec.Card) {
		if increase := g.effects.Rules.SpellCostIncrease(color); increase > 0 {
			total.Generic += increase
			break
		}
	}
	for _, tax := range g.effects.Rules.spellCostTaxes {
		if tax.applies(g, spec.Controller, spec.Card) {
			addActionManaCost(&total, tax.increase)
		}
	}
	for _, color := range cardColors(spec.Card) {
		if reduction := g.effects.Rules.SpellCostReduction(color); reduction > 0 {
			total.Generic = max(0, total.Generic-reduction)
			break
		}
	}
	for _, cardType := range spec.Card.Types() {
		if reduction := g.effects.Rules.SpellTypeCostReduction(cardType); reduction > 0 {
			total.Generic = max(0, total.Generic-reduction)
		}
	}
	total.Generic -= computeConditionalCostReduction(g, spec.Controller, spec.Card, total.Generic, spec.Targets)
	lifeLoss = spec.LifeLoss
	if spec.ChannelGeneric {
		lifeLoss += total.Generic
		total.Generic = 0
	}
	if spec.AnyColor {
		total = flattenManaCost(total)
	}
	return total, lifeLoss
}

func directSpellManaCosts(costs []Cost, xValue int) (ManaCost, bool) {
	var total ManaCost
	for _, cost := range costs {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			addActionManaCost(&total, resolvedSpellManaCost(payment.MC, xValue))
		case *xManaCost:
			total.Generic += xValue
		default:
			return ManaCost{}, false
		}
	}
	return total, true
}

func resolvedSpellManaCost(cost ManaCost, xValue int) ManaCost {
	if cost.HasX {
		cost.Generic += xValue * cost.XCount
		cost.HasX = false
		cost.XCount = 0
	}
	return cost
}

func flattenManaCost(cost ManaCost) ManaCost {
	return ManaCost{Generic: manaCostUnits(cost)}
}

func lockPaymentCosts(costs []Cost, sourceID, controller uuid.UUID, targets []uuid.UUID, xValue int, validation *Game, chooser Player) ([]Cost, map[uuid.UUID]bool, error) {
	return lockPaymentCostsWithManaPolicy(costs, sourceID, controller, targets, xValue, validation, chooser, true)
}

func canPayWithManaPolicy(cost Cost, sourceID, controller uuid.UUID, g *Game, reduceActivation bool) bool {
	if mana, ok := cost.(*ManaCostPayment); ok && !reduceActivation {
		return g.CanAfford(controller, mana.MC, nil)
	}
	if either, ok := cost.(*eitherCost); ok {
		if either.locked {
			return either.selected >= 0 && either.selected < len(either.options) && canPayWithManaPolicy(either.options[either.selected], sourceID, controller, g, reduceActivation)
		}
		for _, option := range either.options {
			if canPayWithManaPolicy(option, sourceID, controller, g, reduceActivation) {
				return true
			}
		}
		return false
	}
	return cost.CanPay(sourceID, controller, g)
}

func lockPaymentCostsWithManaPolicy(costs []Cost, sourceID, controller uuid.UUID, targets []uuid.UUID, xValue int, validation *Game, chooser Player, reduceActivation bool) ([]Cost, map[uuid.UUID]bool, error) {
	var locked []Cost
	outcomes := make(map[uuid.UUID]bool)
	var lock func(Cost) error
	lock = func(cost Cost) error {
		switch choice := cost.(type) {
		case *eitherCost:
			var payable []Cost
			var labels []string
			for _, option := range choice.options {
				if canPayWithManaPolicy(option, sourceID, controller, validation, reduceActivation) {
					payable = append(payable, option)
					labels = append(labels, option.Text())
				}
			}
			if len(payable) == 0 {
				return fmt.Errorf("cannot pay additional cost: %s", choice.Text())
			}
			selected := 0
			if len(payable) > 1 {
				selected = chooser.ChooseMode(labels, choice.Text())
				if selected < 0 || selected >= len(payable) {
					selected = 0
				}
			}
			return lock(payable[selected])
		case *optionalCost:
			paid := choice.inner != nil && choice.inner.CanPay(sourceID, controller, validation) && chooser.ChooseMayAbility(choice.prompt)
			if choice.result == nil {
				outcomes[sourceID] = paid
			} else {
				locked = append(locked, &optionalOutcomeCost{result: choice.result, paid: paid})
			}
			if paid {
				return lock(choice.inner)
			}
			return nil
		default:
			locked = append(locked, cost)
			return nil
		}
	}
	for _, cost := range costs {
		if contextual, ok := cost.(contextualActionCost); ok {
			cost = contextual.resolveActionCost(actionCostContext{Targets: targets, XValue: xValue})
		}
		if err := lock(cost); err != nil {
			return nil, nil, err
		}
	}
	return locked, outcomes, nil
}

// Commit moves the proposed card out of its source zone, activates the exact
// prevalidated mana plan, and pays the locked total cost once.
func (tx *spellPaymentTransaction) Commit() error {
	if tx.game.removeCardFromZone(tx.controller, tx.card.ID(), tx.zone) == nil {
		return fmt.Errorf("could not remove %s from %s", tx.card.Name(), tx.zone)
	}
	tx.game.resolution.SetLastCostReveal(nil)
	return tx.payment.Commit()
}
