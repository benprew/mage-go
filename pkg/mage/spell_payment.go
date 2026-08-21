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
	game             *Game
	card             Card
	controller       uuid.UUID
	zone             Zone
	mana             ManaCost
	lifeLoss         int
	costs            []Cost
	optionalOutcomes map[uuid.UUID]bool
	spellContext     *SpellPaymentContext
	manaSolution     *ManaSolution
}

type spellPaymentSpec struct {
	Card       Card
	Controller uuid.UUID
	Zone       Zone
	Mana       ManaCost
	LifeLoss   int
	Costs      []Cost
	Targets    []uuid.UUID
	XValue     int
	AnyColor   bool
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
		if spec.AnyColor {
			totalMana = flattenManaCost(totalMana)
		}
		spellCtx := SpellContextForCard(spec.Card)
		solution, err := g.planManaForCost(spec.Controller, totalMana, AutoTapHint{CastingCard: spec.Card.ID()}, spellCtx)
		if err != nil {
			return nil, fmt.Errorf("cannot pay total mana cost %s for %s: %w", totalMana, spec.Card.Name(), err)
		}
		return &spellPaymentTransaction{
			game:         g,
			card:         spec.Card,
			controller:   spec.Controller,
			zone:         spec.Zone,
			mana:         totalMana,
			lifeLoss:     spec.LifeLoss,
			spellContext: spellCtx,
			manaSolution: solution,
		}, nil
	}

	validation := cloneGameForSpellPayment(g)
	validation.currentX = spec.XValue
	if validation.removeCardFromZone(spec.Controller, spec.Card.ID(), spec.Zone) == nil {
		return nil, fmt.Errorf("could not move %s from %s while validating its total cost", spec.Card.Name(), spec.Zone)
	}

	lockedCosts, outcomes, err := lockSpellCosts(spec.Costs, spec.Card.ID(), spec.Controller, spec.Targets, spec.XValue, validation, player)
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
	if spec.AnyColor {
		totalMana = flattenManaCost(totalMana)
	}

	spellCtx := SpellContextForCard(spec.Card)
	solution, err := g.planManaForCost(spec.Controller, totalMana, AutoTapHint{CastingCard: spec.Card.ID()}, spellCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot pay total mana cost %s for %s: %w", totalMana, spec.Card.Name(), err)
	}
	if err := validateSpellPayment(validation, spec, totalMana, nonMana, solution, spellCtx); err != nil {
		return nil, err
	}

	return &spellPaymentTransaction{
		game:             g,
		card:             spec.Card,
		controller:       spec.Controller,
		zone:             spec.Zone,
		mana:             totalMana,
		lifeLoss:         spec.LifeLoss,
		costs:            nonMana,
		optionalOutcomes: outcomes,
		spellContext:     spellCtx,
		manaSolution:     solution,
	}, nil
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

func cloneGameForSpellPayment(g *Game) *Game {
	wasBattlefieldShared := g.battlefieldShared
	wasBattlefieldSliceShared := g.battlefieldSliceShared
	ownedPermanents := g.ownedPermanents
	clone := g.Clone()
	g.battlefieldShared = wasBattlefieldShared
	g.battlefieldSliceShared = wasBattlefieldSliceShared
	g.ownedPermanents = ownedPermanents

	clone.battlefield = make([]*Permanent, len(g.battlefield))
	for i, permanent := range g.battlefield {
		copy := &Permanent{}
		clonePermanentInto(copy, permanent)
		clone.battlefield[i] = copy
	}
	clone.battlefieldShared = false
	clone.battlefieldSliceShared = false
	clone.ownedPermanents = nil
	return clone
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

func lockSpellCosts(costs []Cost, sourceID, controller uuid.UUID, targets []uuid.UUID, xValue int, validation *Game, chooser Player) ([]Cost, map[uuid.UUID]bool, error) {
	var locked []Cost
	outcomes := make(map[uuid.UUID]bool)
	var lock func(Cost) error
	lock = func(cost Cost) error {
		switch choice := cost.(type) {
		case *eitherCost:
			var payable []Cost
			var labels []string
			for _, option := range choice.options {
				if option.CanPay(sourceID, controller, validation) {
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
			outcomes[sourceID] = false
			if choice.inner.CanPay(sourceID, controller, validation) && chooser.ChooseMayAbility(choice.prompt) {
				outcomes[sourceID] = true
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

func validateSpellPayment(validation *Game, spec spellPaymentSpec, mana ManaCost, costs []Cost, solution *ManaSolution, spellCtx *SpellPaymentContext) error {
	player := validation.GetPlayer(spec.Controller)
	if player == nil {
		return ErrPlayerNotFound
	}
	if err := validation.applyManaSolution(spec.Controller, solution); err != nil {
		return fmt.Errorf("cannot activate mana abilities for %s: %w", spec.Card.Name(), err)
	}
	if !mana.IsZero() {
		if err := player.ManaPool().Pay(mana, spellCtx); err != nil {
			return fmt.Errorf("cannot pay total mana cost %s for %s: %w", mana, spec.Card.Name(), err)
		}
	}
	if spec.LifeLoss > 0 {
		validation.PlayerLoseLife(player, spec.LifeLoss)
	}
	for _, cost := range costs {
		validationCost := cost
		if randomDiscard, ok := cost.(*discardRandomCost); ok {
			validationCost = &discardCost{amount: randomDiscard.amount}
		}
		if !validationCost.CanPay(spec.Card.ID(), spec.Controller, validation) {
			return fmt.Errorf("cannot pay total cost for %s: %s", spec.Card.Name(), cost.Text())
		}
		if err := validationCost.Pay(spec.Card.ID(), spec.Controller, validation); err != nil {
			return fmt.Errorf("cannot pay total cost for %s: %w", spec.Card.Name(), err)
		}
	}
	return nil
}

// Commit moves the proposed card out of its source zone, activates the exact
// prevalidated mana plan, and pays the locked total cost once.
func (tx *spellPaymentTransaction) Commit() error {
	if tx.game.removeCardFromZone(tx.controller, tx.card.ID(), tx.zone) == nil {
		return fmt.Errorf("could not remove %s from %s", tx.card.Name(), tx.zone)
	}
	player := tx.game.GetPlayer(tx.controller)
	if player == nil {
		return ErrPlayerNotFound
	}
	tx.game.lastCostReveal = nil
	if err := tx.game.applyManaSolution(tx.controller, tx.manaSolution); err != nil {
		return err
	}
	player.ManaPool().ResetLastDrained()
	if !tx.mana.IsZero() {
		if err := player.ManaPool().Pay(tx.mana, tx.spellContext); err != nil {
			return err
		}
	}
	if tx.lifeLoss > 0 {
		tx.game.PlayerLoseLife(player, tx.lifeLoss)
	}
	for _, cost := range tx.costs {
		if err := cost.Pay(tx.card.ID(), tx.controller, tx.game); err != nil {
			return err
		}
	}
	for sourceID, paid := range tx.optionalOutcomes {
		tx.game.setOptionalCostPaid(sourceID, paid)
	}
	return nil
}
