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
	wasBattlefieldShared := g.battlefieldShared
	wasBattlefieldSliceShared := g.battlefieldSliceShared
	ownedPermanents := g.ownedPermanents
	clone := g.Clone()
	g.battlefieldShared = wasBattlefieldShared
	g.battlefieldSliceShared = wasBattlefieldSliceShared
	g.ownedPermanents = ownedPermanents

	clone.battlefield = make([]*Permanent, len(g.battlefield))
	for i, permanent := range g.battlefield {
		clonedPermanent := &Permanent{}
		clonePermanentInto(clonedPermanent, permanent)
		clonedPermanent.RuntimeAbilities = clonePaymentAbilities(permanent.RuntimeAbilities)
		clone.battlefield[i] = clonedPermanent
	}
	clone.battlefieldShared = false
	clone.battlefieldSliceShared = false
	clone.ownedPermanents = nil
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
	solution, err := g.planManaForCost(spec.Controller, spec.Mana, spec.Hint, spec.SpellContext)
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
	if err := validation.applyManaSolution(tx.controller, tx.manaSolution); err != nil {
		return fmt.Errorf("cannot activate mana abilities: %w", err)
	}
	if !tx.mana.IsZero() {
		if err := player.ManaPool().Pay(tx.mana, tx.spellContext); err != nil {
			return err
		}
	}
	if tx.lifeLoss > 0 {
		if player.Life() <= tx.lifeLoss {
			return fmt.Errorf("cannot pay %d life", tx.lifeLoss)
		}
		validation.PlayerLoseLife(player, tx.lifeLoss)
	}
	for _, cost := range tx.costs {
		validationCost := cost
		if randomDiscard, ok := cost.(*discardRandomCost); ok {
			validationCost = &discardCost{amount: randomDiscard.amount}
		}
		if !validationCost.CanPay(tx.sourceID, tx.controller, validation) {
			return fmt.Errorf("cannot pay total cost: %s", cost.Text())
		}
		if err := validationCost.Pay(tx.sourceID, tx.controller, validation); err != nil {
			return fmt.Errorf("cannot pay total cost: %w", err)
		}
	}
	return nil
}

func (tx *costPaymentTransaction) Commit() error {
	player := tx.game.GetPlayer(tx.controller)
	if player == nil {
		return ErrPlayerNotFound
	}
	if err := tx.game.applyManaSolution(tx.controller, tx.manaSolution); err != nil {
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
		if err := cost.Pay(tx.sourceID, tx.controller, tx.game); err != nil {
			return err
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
