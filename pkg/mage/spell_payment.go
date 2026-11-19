package mage

import (
	"fmt"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// spellPaymentTransaction is the locked CR 601.2f total cost for one spell.
// prepareSpellPaymentTransaction resolves branching costs, combines every mana
// component, and proves the complete payment on a game clone before Commit
// mutates live state.
type spellPaymentTransaction struct {
	game       *Game
	card       *Card
	controller uuid.UUID
	zone       Zone
	payment    *costPaymentTransaction
}

type spellPaymentSpec struct {
	Card       *Card
	Controller uuid.UUID
	Zone       Zone
	Mana       ManaCost
	LifeLoss   int
	Costs      []Cost
	Targets    []uuid.UUID
	XValue     int
	AnyColor   bool
}

func (g *Game) spellGenericCostAdjustment(controller uuid.UUID, card *Card, targets []uuid.UUID, includePossibleReductions bool) int {
	rules := g.effects.Rules
	increase, reduction := 0, 0
	if len(rules.SpellCostIncreases) > 0 || len(rules.SpellCostReductions) > 0 {
		for _, color := range card.ManaCost().Colors() {
			if increase == 0 {
				increase = max(0, rules.SpellCostIncrease(color))
			}
			if reduction == 0 {
				reduction = max(0, rules.SpellCostReduction(color))
			}
		}
	}
	if len(rules.SpellTypeCostReductions) > 0 {
		for _, kind := range card.Types() {
			reduction += max(0, rules.SpellTypeCostReduction(kind))
		}
	}
	reduction += applySelfCostReductions(g, controller, card, targets, includePossibleReductions)
	reduction += applyExternalCostReductions(g, controller, card)
	return increase - reduction
}

func (g *Game) spellAnnouncementCosts(spec spellPaymentSpec) spellPaymentSpec {
	spec.Mana = resolvedSpellManaCost(spec.Mana, spec.XValue)
	spec.Mana.Generic = max(0, spec.Mana.Generic+g.spellGenericCostAdjustment(spec.Controller, spec.Card, spec.Targets, false))
	if g.effects.Rules.IsChannelActive(spec.Controller) {
		spec.LifeLoss += spec.Mana.Generic
		spec.Mana.Generic = 0
	}

	costs := append([]Cost(nil), spec.Costs...)
	costs = append(costs, spec.Card.AdditionalCosts()...)
	for _, ability := range spec.Card.abilities {
		if spell, ok := ability.(*SpellAbility); ok && spell.Kind() == ActionSpell {
			costs = append(costs, spell.Costs()...)
		}
	}
	spec.Costs = g.prepareActionCosts(costs, spec.Targets, spec.XValue)
	return spec
}

func (g *Game) canAffordSpellAnnouncement(player Player, card *Card, x int, targets []uuid.UUID) bool {
	spec := g.spellAnnouncementCosts(spellPaymentSpec{
		Card: card, Controller: player.PlayerID(), Mana: card.ManaCost(), XValue: x, Targets: targets,
	})
	life := spec.LifeLoss
	for _, cost := range spec.Costs {
		life += LifePaymentAmount(cost)
	}
	if life > 0 && player.Life() <= life {
		return false
	}
	extra, _ := directSpellManaCosts(spec.Costs, x)
	addActionManaCost(&spec.Mana, extra)
	return g.CanAfford(spec.Controller, spec.Mana, SpellContextForCard(card))
}

func (g *Game) prepareSpellPaymentTransaction(spec spellPaymentSpec) (*spellPaymentTransaction, error) {
	if spec.Card == nil {
		return nil, ErrSourceNotFound
	}
	player := g.GetPlayer(spec.Controller)
	if player == nil {
		return nil, ErrPlayerNotFound
	}
	spec = g.spellAnnouncementCosts(spec)
	if extraMana, direct := directSpellManaCosts(spec.Costs, spec.XValue); direct {
		totalMana := spec.Mana
		addActionManaCost(&totalMana, extraMana)
		if spec.AnyColor {
			totalMana = flattenManaCost(totalMana)
		}
		spellCtx := SpellContextForCard(spec.Card)
		payment, err := g.prepareCostPaymentTransaction(costPaymentSpec{
			Controller:             spec.Controller,
			SourceID:               spec.Card.ID(),
			Mana:                   totalMana,
			LifeLoss:               spec.LifeLoss,
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

	validation := g.Clone()
	validation.resolution.SetX(spec.XValue)
	if validation.removeCardFromZone(spec.Controller, spec.Card.ID(), spec.Zone) == nil {
		return nil, fmt.Errorf("could not move %s from %s while validating its total cost", spec.Card.Name(), spec.Zone)
	}

	lockedCosts, outcomes, err := lockPaymentCosts(spec.Costs, spec.Card.ID(), spec.Controller, spec.Targets, spec.XValue, validation, player)
	if err != nil {
		return nil, err
	}
	totalMana := spec.Mana
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
	payment, err := g.prepareCostPaymentTransaction(costPaymentSpec{
		Controller:             spec.Controller,
		SourceID:               spec.Card.ID(),
		Mana:                   totalMana,
		LifeLoss:               spec.LifeLoss,
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

func directSpellManaCosts(costs []Cost, xValue int) (ManaCost, bool) {
	var total ManaCost
	direct := true
	for _, cost := range costs {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			addActionManaCost(&total, resolvedSpellManaCost(payment.MC, xValue))
		case *xManaCost:
			total.Generic += xValue
		default:
			direct = false
		}
	}
	return total, direct
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
	return ManaCost{Generic: cost.CMC() - cost.Colorless, Colorless: cost.Colorless}
}

func lockPaymentCosts(costs []Cost, sourceID, controller uuid.UUID, targets []uuid.UUID, xValue int, validation *Game, chooser Player) ([]Cost, map[uuid.UUID]bool, error) {
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

// Commit moves the proposed card out of its source zone, activates the exact
// prevalidated mana plan, and pays the locked total cost once.
func (tx *spellPaymentTransaction) Commit() error {
	if tx.game.removeCardFromZone(tx.controller, tx.card.ID(), tx.zone) == nil {
		return fmt.Errorf("could not remove %s from %s", tx.card.Name(), tx.zone)
	}
	tx.game.resolution.SetLastCostReveal(nil)
	return tx.payment.Commit()
}
