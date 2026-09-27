package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// costPaymentTransaction is a locked total cost shared by spell casting,
// activated abilities, attack costs, and resolution-time mana payments.
// Preparation plans mana production and proves the complete payment on an
// isolated game. Commit performs that same payment once on live state.
type costPaymentTransaction struct {
	game                   *Game
	controller             uuid.UUID
	sourceID               uuid.UUID
	mana                   ManaCost
	lifeLoss               int
	costs                  []Cost
	optionalOutcomes       map[uuid.UUID]bool
	spellContext           *SpellPaymentContext
	manaSolution           *ManaSolution
	resetDrainedBeforeMana bool
	paidNumbers            map[*NumberBinding]int
	paidCards              map[*CardBinding]capturedCard
}

type costPaymentSpec struct {
	Controller             uuid.UUID
	SourceID               uuid.UUID
	Mana                   ManaCost
	LifeLoss               int
	Costs                  []Cost
	OptionalOutcomes       map[uuid.UUID]bool
	Hint                   AutoTapHint
	SpellContext           *SpellPaymentContext
	ResetDrainedBeforeMana bool
}

func cloneGameForCostPayment(g *Game) *Game {
	wasBattlefieldShared := g.zones.battlefieldShared
	wasBattlefieldSliceShared := g.zones.battlefieldSliceShared
	ownedPermanents := g.zones.ownedPermanents
	clone := g.Clone()
	g.zones.battlefieldShared = wasBattlefieldShared
	g.zones.battlefieldSliceShared = wasBattlefieldSliceShared
	g.zones.ownedPermanents = ownedPermanents

	clone.zones.battlefield = make([]*Permanent, len(g.zones.battlefield))
	for i, permanent := range g.zones.battlefield {
		clonedPermanent := &Permanent{}
		clonePermanentInto(clonedPermanent, permanent)
		clonedPermanent.RuntimeAbilities = clonePaymentAbilities(permanent.RuntimeAbilities)
		clone.zones.battlefield[i] = clonedPermanent
	}
	clone.zones.battlefieldShared = false
	clone.zones.battlefieldSliceShared = false
	clone.zones.ownedPermanents = nil
	return clone
}

func clonePaymentAbilities(abilities []Ability) []Ability {
	cloned := make([]Ability, len(abilities))
	for i, ability := range abilities {
		cloned[i] = clonePaymentAbility(ability)
	}
	return cloned
}

func clonePaymentAbility(ability Ability) Ability {
	if granted, ok := ability.(*grantedByEffect); ok {
		return &grantedByEffect{
			Ability:                clonePaymentAbility(granted.Ability),
			intrinsicBasicLandMana: granted.intrinsicBasicLandMana,
		}
	}
	if action, ok := ability.(*ActionDefinition); ok {
		cloned := *action
		return &cloned
	}
	return ability
}

func (g *Game) prepareCostPaymentTransaction(spec costPaymentSpec, validation *Game) (*costPaymentTransaction, error) {
	if g.GetPlayer(spec.Controller) == nil {
		return nil, ErrPlayerNotFound
	}
	solution, err := g.mana.PlanManaForCost(g, spec.Controller, spec.Mana, spec.Hint, spec.SpellContext)
	if err != nil {
		return nil, err
	}
	tx := &costPaymentTransaction{
		game:                   g,
		controller:             spec.Controller,
		sourceID:               spec.SourceID,
		mana:                   spec.Mana,
		lifeLoss:               spec.LifeLoss,
		costs:                  append([]Cost(nil), spec.Costs...),
		optionalOutcomes:       spec.OptionalOutcomes,
		spellContext:           spec.SpellContext,
		manaSolution:           solution,
		resetDrainedBeforeMana: spec.ResetDrainedBeforeMana,
	}
	if validation == nil {
		validation = cloneGameForCostPayment(g)
	}
	if err := tx.validate(validation); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *costPaymentTransaction) validate(validation *Game) error {
	player := validation.GetPlayer(tx.controller)
	if player == nil {
		return ErrPlayerNotFound
	}
	if err := validation.mana.ApplyManaSolution(validation, tx.controller, tx.manaSolution); err != nil {
		return fmt.Errorf("cannot activate mana abilities: %w", err)
	}
	if !tx.mana.IsZero() {
		if err := player.ManaPool().Pay(tx.mana, tx.spellContext); err != nil {
			return err
		}
	}
	if tx.lifeLoss > 0 {
		if player.Life() < tx.lifeLoss {
			return fmt.Errorf("cannot pay %d life", tx.lifeLoss)
		}
		validation.PlayerLoseLife(player, tx.lifeLoss)
	}
	for i, cost := range tx.costs {
		validationCost := cost
		if discard, ok := cost.(*discardCost); ok {
			locked := *discard
			locked.selected = nil
			locked.locked = true
			if discard.amount > 0 {
				payer := tx.game.GetPlayer(tx.controller)
				if payer == nil {
					return ErrPlayerNotFound
				}
				for _, card := range payer.ChooseCardsFromHand(discard.amount, "discard cost", tx.game) {
					if card != nil {
						locked.selected = append(locked.selected, card.ID())
					}
				}
			}
			validationCost = &locked
		}
		if randomDiscard, ok := cost.(*discardRandomCost); ok {
			validationCost = &discardCost{amount: randomDiscard.amount}
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok {
			sacCopy := *sacrifice
			sacCopy.selected = make([]capturedPermanent, 0, sacCopy.count)
			sacCopy.chooser = tx.game.GetPlayer(tx.controller)
			validationCost = &sacCopy
		}
		if counter, ok := cost.(*removeAnyCountersCost); ok {
			cntCopy := *counter
			source := tx.game.FindPermanent(tx.sourceID)
			if source == nil {
				return fmt.Errorf("cannot pay chosen counter cost: source absent")
			}
			payer := tx.game.GetPlayer(tx.controller)
			cntCopy.count = payer.ChooseNumber(0, int(source.Counters[cntCopy.ct]), cntCopy.Text())
			cntCopy.incarnation = source.incarnationID
			cntCopy.locked = true
			validationCost = &cntCopy
			tx.costs[i] = &cntCopy
		}
		if !validationCost.CanPay(tx.sourceID, tx.controller, validation) {
			return fmt.Errorf("cannot pay total cost: %s", cost.Text())
		}
		if err := validationCost.Pay(tx.sourceID, tx.controller, validation); err != nil {
			return fmt.Errorf("cannot pay total cost: %w", err)
		}
		if sacrifice, ok := validationCost.(*sacrificeMatchingCost); ok {
			sacrifice.locked = true
			sacrifice.chooser = nil
			tx.costs[i] = sacrifice
		}
		if _, ok := cost.(*discardCost); ok {
			tx.costs[i] = validationCost
		}
	}
	return nil
}

func (tx *costPaymentTransaction) Commit() (commitErr error) {
	tx.paidNumbers = nil
	tx.paidCards = nil
	defer func() {
		if commitErr != nil {
			tx.paidNumbers = nil
			tx.paidCards = nil
		}
	}()
	player := tx.game.GetPlayer(tx.controller)
	if player == nil {
		return ErrPlayerNotFound
	}
	for _, cost := range tx.costs {
		if discard, ok := cost.(*discardCost); ok && !discard.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen discard")
		}
	}
	if err := tx.game.mana.ApplyManaSolution(tx.game, tx.controller, tx.manaSolution); err != nil {
		return err
	}
	if tx.resetDrainedBeforeMana {
		player.ManaPool().ResetLastDrained()
	}
	if !tx.mana.IsZero() {
		if err := player.ManaPool().Pay(tx.mana, tx.spellContext); err != nil {
			return err
		}
	}
	if tx.lifeLoss > 0 {
		tx.game.PlayerLoseLife(player, tx.lifeLoss)
	}
	for _, cost := range tx.costs {
		var paidPower, paidToughness, paidManaValue int
		var paidCardID uuid.UUID
		var discardedCardID uuid.UUID
		if discard, ok := cost.(*discardCost); ok && discard.captureCard && len(discard.selected) == 1 {
			discardedCardID = discard.selected[0]
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureCard) {
			if !sacrifice.CanPay(tx.sourceID, tx.controller, tx.game) || len(sacrifice.selected) != 1 {
				return fmt.Errorf("cannot pay chosen sacrifice")
			}
			chosen := tx.game.FindPermanent(sacrifice.selected[0].id)
			if sacrifice.captureCard && !chosen.IsToken {
				paidCardID = chosen.ID()
			}
			if sacrifice.captureStats {
				paidPower = chosen.CurrentPower(tx.game)
				paidToughness = chosen.CurrentToughness(tx.game)
			}
			if sacrifice.captureManaValue {
				paidManaValue = chosen.Card.ManaCost().CMC()
			}
		}
		if err := cost.Pay(tx.sourceID, tx.controller, tx.game); err != nil {
			return err
		}
		if discard, ok := cost.(*discardCost); ok && discard.captureCard {
			if tx.paidCards == nil {
				tx.paidCards = make(map[*CardBinding]capturedCard)
			}
			tx.paidCards[discard.paidCard] = capturedCard{id: discardedCardID, visit: tx.game.graveyardVisit(discardedCardID), zone: ZoneGraveyard}
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && sacrifice.captureCard {
			if tx.paidCards == nil {
				tx.paidCards = make(map[*CardBinding]capturedCard)
			}
			tx.paidCards[sacrifice.paidCard] = capturedCard{id: paidCardID, visit: tx.game.graveyardVisit(paidCardID), zone: ZoneGraveyard}
		}
		if life, ok := cost.(*lifePayCost); ok && life.capture && life.paid != nil {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			tx.paidNumbers[life.paid] = life.amount
		}
		if counter, ok := cost.(*removeAnyCountersCost); ok {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			tx.paidNumbers[counter.paid] = counter.count
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue) {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			if sacrifice.captureStats {
				tx.paidNumbers[sacrifice.paidPower] = paidPower
				tx.paidNumbers[sacrifice.paidToughness] = paidToughness
			}
			if sacrifice.captureManaValue {
				tx.paidNumbers[sacrifice.paidManaValue] = paidManaValue
			}
		}
	}
	for sourceID, paid := range tx.optionalOutcomes {
		tx.game.setOptionalCostPaid(sourceID, paid)
	}
	return nil
}

type actionPaymentSpec struct {
	Controller               uuid.UUID
	SourceID                 uuid.UUID
	Costs                    []Cost
	Targets                  []uuid.UUID
	XValue                   int
	Hint                     AutoTapHint
	ApplyActivationReduction bool
}

func (g *Game) prepareActionPaymentTransaction(spec actionPaymentSpec) (*costPaymentTransaction, error) {
	player := g.GetPlayer(spec.Controller)
	if player == nil {
		return nil, ErrPlayerNotFound
	}
	validation := cloneGameForCostPayment(g)
	locked, outcomes, err := lockPaymentCosts(spec.Costs, spec.SourceID, spec.Controller, spec.Targets, spec.XValue, validation, player)
	if err != nil {
		return nil, err
	}

	var mana ManaCost
	var nonMana []Cost
	for _, cost := range locked {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			component := payment.MC
			if spec.ApplyActivationReduction {
				component = payment.reducedCost(spec.SourceID, g)
			}
			if component.HasX {
				component.Generic += spec.XValue * max(1, component.XCount)
				component.HasX = false
				component.XCount = 0
			}
			addActionManaCost(&mana, component)
		case *xManaCost:
			mana.Generic += spec.XValue
		default:
			nonMana = append(nonMana, cost)
		}
	}

	return g.prepareCostPaymentTransaction(costPaymentSpec{
		Controller:       spec.Controller,
		SourceID:         spec.SourceID,
		Mana:             mana,
		Costs:            nonMana,
		OptionalOutcomes: outcomes,
		Hint:             spec.Hint,
	}, validation)
}
