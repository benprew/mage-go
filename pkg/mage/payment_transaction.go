package mage

import (
	"fmt"
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// costPaymentTransaction is a locked total cost shared by spell casting,
// activated abilities, attack costs, and resolution-time mana payments.
// Preparation plans mana production and proves the complete payment on an
// isolated game. Commit performs that same payment once on live state.
type costPaymentTransaction struct {
	paidXMana              [AnyColor + 1]int
	manaX                  int
	paidMana               [AnyColor + 1]int
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
	revealedHandCards      []capturedCard
	paidCardSets           map[*CardsBinding][]capturedCard
	paidPermanents         map[*PermanentBinding]capturedPermanent
	paidTaps               []capturedPermanent
}

type costPaymentSpec struct {
	ManaX                  int
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
	// Random costs must follow deterministic costs (CR 601.2h).
	ordered := make([]Cost, 0, len(spec.Costs))
	for _, cost := range spec.Costs {
		if _, random := cost.(*discardRandomCost); !random {
			ordered = append(ordered, cost)
		}
	}
	for _, cost := range spec.Costs {
		if _, random := cost.(*discardRandomCost); random {
			ordered = append(ordered, cost)
		}
	}
	tx := &costPaymentTransaction{
		manaX:                  spec.ManaX,
		game:                   g,
		controller:             spec.Controller,
		sourceID:               spec.SourceID,
		mana:                   spec.Mana,
		lifeLoss:               spec.LifeLoss,
		costs:                  ordered,
		optionalOutcomes:       spec.OptionalOutcomes,
		spellContext:           spec.SpellContext,
		manaSolution:           solution,
		resetDrainedBeforeMana: spec.ResetDrainedBeforeMana,
	}
	for i, cost := range tx.costs {
		if sacrifice, ok := cost.(*pluralSacrificeCost); ok && sacrifice.mandatorySource {
			p := g.FindPermanent(tx.sourceID)
			if p == nil || p.PhasedOut {
				return nil, ErrSourceNotFound
			}
			locked := *sacrifice
			locked.sourceVisit = capturedPermanent{p.ID(), p.incarnationID}
			tx.costs[i] = &locked
			continue
		}
		source, ok := cost.(*sourceCardCost)
		if !ok {
			continue
		}
		p := g.FindPermanent(tx.sourceID)
		if p == nil || p.PhasedOut {
			return nil, ErrSourceNotFound
		}
		locked := *source
		locked.incarnation = p.incarnationID
		tx.costs[i] = &locked
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
		if player.Life() < tx.lifeLoss || !validation.CanLoseLife(tx.controller) {
			return fmt.Errorf("cannot pay %d life", tx.lifeLoss)
		}
		validation.PlayerLoseLife(player, tx.lifeLoss)
	}
	for i := 0; i < len(tx.costs); i++ {
		cost := tx.costs[i]
		if optional, ok := cost.(*optionalCost); ok {
			_, directX := optional.inner.(*removeXCountersCost)
			fixed, directFixed := optional.inner.(*removeCountersCost)
			_, alternative := optional.inner.(*eitherCost)
			counterAlternatives := alternative && (uncapturedFixedCounterAlternatives(optional.inner) || uncapturedXCounterAlternatives(optional.inner))
			if directX || (directFixed && !fixed.capture) || counterAlternatives {
				// Lock acceptance before the shared counter payment checks the source.
				locked, outcomes, err := lockPaymentCosts([]Cost{optional}, tx.sourceID, tx.controller, nil, tx.game.resolution.X(), validation, tx.game.GetPlayer(tx.controller))
				if err != nil {
					return err
				}
				costs := append([]Cost(nil), tx.costs[:i]...)
				costs = append(costs, locked...)
				tx.costs = append(costs, tx.costs[i+1:]...)
				if len(outcomes) != 0 {
					copied := make(map[uuid.UUID]bool, len(tx.optionalOutcomes)+len(outcomes))
					maps.Copy(copied, tx.optionalOutcomes)
					maps.Copy(copied, outcomes)
					tx.optionalOutcomes = copied
				}
				i--
				continue
			}
		}
		if counter, ok := cost.(*removeXCountersCost); ok {
			cost = counter.resolveActionCost(actionCostContext{XValue: tx.game.resolution.X()})
			tx.costs[i] = cost
		}
		validationCost := cost
		if either, ok := cost.(*eitherCost); ok {
			locked := *either.resolveActionCost(actionCostContext{XValue: tx.game.resolution.X()}).(*eitherCost)
			var payable []int
			var labels []string
			for index, option := range locked.options {
				if option.CanPay(tx.sourceID, tx.controller, validation) {
					payable = append(payable, index)
					labels = append(labels, option.Text())
				}
			}
			if len(payable) == 0 {
				return fmt.Errorf("no payable cost option")
			}
			choice := 0
			if len(payable) > 1 {
				choice = tx.game.GetPlayer(tx.controller).ChooseMode(labels, either.Text())
				if choice < 0 || choice >= len(payable) {
					return fmt.Errorf("invalid cost option")
				}
			}
			locked.selected = payable[choice]
			locked.locked = true
			if nested, ok := locked.options[locked.selected].(*eitherCost); ok {
				tx.costs[i] = nested
				i--
				continue
			}
			if optional, ok := locked.options[locked.selected].(*optionalCost); ok {
				if uncapturedFixedCounterAlternatives(optional.inner) || uncapturedXCounterAlternatives(optional.inner) {
					tx.costs[i] = optional
					i--
					continue
				}
			}
			if exile, ok := locked.options[locked.selected].(*exileFromGraveyardCost); ok {
				chosen := *exile
				if err := chosen.choose(tx.sourceID, tx.controller, tx.game); err != nil {
					return err
				}
				locked.options[locked.selected] = &chosen
			}
			validationCost = &locked
			tx.costs[i] = &locked
			if placement, ok := locked.options[locked.selected].(*putCounterOnMatchingCost); ok {
				cost = placement
				validationCost = placement
				tx.costs[i] = placement
			}
			if counter, ok := locked.options[locked.selected].(*removeCountersCost); ok {
				cost = counter
				validationCost = counter
				tx.costs[i] = counter
			}
			if ante, ok := locked.options[locked.selected].(*anteCost); ok {
				cost = ante
				validationCost = ante
				tx.costs[i] = ante
			}
		}
		if ante, ok := cost.(*anteCost); ok {
			locked := ante.lock(tx.controller, validation)
			validationCost = locked
			tx.costs[i] = locked
		}
		if reveal, ok := cost.(*revealFromHandCost); ok {
			payer := tx.game.GetPlayer(tx.controller)
			if payer == nil {
				return ErrPlayerNotFound
			}
			var matches []Card
			for _, card := range payer.Hand() {
				if card.ID() != tx.sourceID && (reveal.filter.IsZero() || reveal.filter.Match(card)) {
					matches = append(matches, card)
				}
			}
			if len(matches) == 0 {
				return fmt.Errorf("cannot pay chosen reveal")
			}
			chosen := matches[0]
			if chooser, ok := payer.(interface{ ChooseFromCards([]Card, string) Card }); ok {
				if selected := chooser.ChooseFromCards(matches, reveal.label); selected != nil {
					chosen = selected
				}
			}
			locked := *reveal
			locked.selected = chosen.ID()
			if visitor, ok := payer.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
				locked.visit = visitor.handVisit(chosen.ID())
			}
			locked.locked = true
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if discard, ok := cost.(*discardCost); ok {
			locked := *discard
			locked.selected = nil
			locked.results = nil
			locked.locked = true
			if discard.amount > 0 {
				payer := tx.game.GetPlayer(tx.controller)
				if payer == nil {
					return ErrPlayerNotFound
				}
				for _, card := range discard.choose(payer, tx.game) {
					if card != nil {
						selected := capturedCard{id: card.ID(), zone: ZoneHand}
						if visitor, ok := payer.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
							selected.visit = visitor.handVisit(card.ID())
						}
						locked.selected = append(locked.selected, selected)
					}
				}
			}
			validationCost = &locked
		}
		if randomDiscard, ok := cost.(*discardRandomCost); ok {
			locked, err := randomDiscard.lock(tx.controller, validation)
			if err != nil {
				return err
			}
			tx.costs[i] = locked
			validationCost = &locked.discard
		}
		if sacrifice, ok := cost.(*pluralSacrificeCost); ok {
			locked := *sacrifice
			locked.selected = append([]capturedPermanent(nil), sacrifice.selected...)
			locked.results = nil
			locked.chooser = tx.game.GetPlayer(tx.controller)
			validationCost = &locked
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok {
			sacCopy := *sacrifice
			sacCopy.selected = make([]capturedPermanent, 0, sacCopy.count)
			sacCopy.chooser = tx.game.GetPlayer(tx.controller)
			validationCost = &sacCopy
		}
		if exile, ok := cost.(*matchingPermanentExileCost); ok {
			locked := *exile
			locked.chooser = tx.game.GetPlayer(tx.controller)
			validationCost = &locked
		}
		if taps, ok := cost.(*tapNOtherMatchingCost); ok {
			locked := *taps
			locked.selected = make([]capturedPermanent, 0, locked.count)
			locked.chooser = tx.game.GetPlayer(tx.controller)
			validationCost = &locked
		}
		if placement, ok := cost.(*putCounterOnMatchingCost); ok {
			locked := *placement
			if !locked.locked {
				if err := locked.choose(tx.controller, validation, tx.game.GetPlayer(tx.controller)); err != nil {
					return err
				}
			}
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if attached, ok := cost.(*tapAttachedPermanentCost); ok {
			locked := *attached
			if !locked.capture(tx.sourceID, tx.game) {
				return fmt.Errorf("cannot pay attached permanent tap cost")
			}
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if mill, ok := cost.(*millCost); ok {
			locked := *mill
			locked.lockedAmount = max(0, validation.ApplyMillModifiers(tx.controller, mill.amount))
			locked.locked = true
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if exile, ok := cost.(*exileFromGraveyardCost); ok {
			locked := *exile
			if err := locked.choose(tx.sourceID, tx.controller, tx.game); err != nil {
				return err
			}
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if counter, ok := cost.(*removeAnyCountersCost); ok {
			if !counter.CanPay(tx.sourceID, tx.controller, tx.game) {
				return fmt.Errorf("cannot pay chosen counter cost")
			}
			cntCopy := *counter
			source := tx.game.FindPermanent(tx.sourceID)
			if source == nil {
				return fmt.Errorf("cannot pay chosen counter cost: source absent")
			}
			payer := tx.game.GetPlayer(tx.controller)
			cntCopy.count = payer.ChooseNumber(0, source.Counters[cntCopy.ct], cntCopy.Text())
			cntCopy.incarnation = source.incarnationID
			cntCopy.locked = true
			validationCost = &cntCopy
			tx.costs[i] = &cntCopy
		}
		if counter, ok := cost.(*removeCountersCost); ok && (!counter.zeroWithoutSource || counter.amount != 0) {
			locked := *counter
			source := tx.game.FindPermanent(tx.sourceID)
			if source == nil {
				return fmt.Errorf("cannot pay counter cost: source absent")
			}
			locked.incarnation = source.incarnationID
			locked.locked = true
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && (exile.capture || exile.captureCard) {
			locked := *exile
			payer := tx.game.GetPlayer(tx.controller)
			var candidates []Card
			for _, card := range payer.Graveyard() {
				if exile.filter.Match(card) {
					candidates = append(candidates, card)
				}
			}
			if len(candidates) == 0 {
				return fmt.Errorf("cannot pay chosen graveyard exile")
			}
			chosen := payer.ChooseCardFromGraveyard(candidates, exile.text, tx.game)
			if chosen == nil {
				chosen = candidates[0]
			}
			locked.selected = chosen.ID()
			locked.visit = tx.game.graveyardVisit(chosen.ID())
			locked.locked = true
			validationCost = &locked
			tx.costs[i] = &locked
		}
		if !validationCost.CanPay(tx.sourceID, tx.controller, validation) {
			return fmt.Errorf("cannot pay total cost: %s", cost.Text())
		}
		if err := validationCost.Pay(tx.sourceID, tx.controller, validation); err != nil {
			return fmt.Errorf("cannot pay total cost: %w", err)
		}
		if sacrifice, ok := validationCost.(*pluralSacrificeCost); ok {
			sacrifice.locked = true
			sacrifice.chooser = nil
			tx.costs[i] = sacrifice
		}
		if sacrifice, ok := validationCost.(*sacrificeMatchingCost); ok {
			sacrifice.locked = true
			sacrifice.chooser = nil
			tx.costs[i] = sacrifice
		}
		if exile, ok := validationCost.(*matchingPermanentExileCost); ok {
			exile.locked = true
			exile.chooser = nil
			tx.costs[i] = exile
		}
		if taps, ok := validationCost.(*tapNOtherMatchingCost); ok {
			taps.locked = true
			taps.chooser = nil
			tx.costs[i] = taps
		}
		if _, ok := cost.(*discardCost); ok {
			tx.costs[i] = validationCost
		}
	}
	return nil
}

func (tx *costPaymentTransaction) Commit() error {
	if tx.manaSolution != nil {
		for _, action := range tx.manaSolution.SourcesToTap {
			if action.PlayerManaAbilityID != uuid.Nil {
				preflight := *tx
				preflight.game = cloneGameForCostPayment(tx.game)
				if err := preflight.commit(); err != nil {
					return err
				}
				break
			}
		}
	}

	tx.paidNumbers = nil
	tx.paidCards = nil
	tx.revealedHandCards = nil
	tx.paidCardSets = nil
	tx.paidPermanents = nil
	tx.paidTaps = nil
	for _, cost := range tx.costs {
		_, random := cost.(*discardRandomCost)
		_, isSacrifice := cost.(*sacrificeMatchingCost)
		_, isPluralSacrifice := cost.(*pluralSacrificeCost)
		_, isPlacement := cost.(*putCounterOnMatchingCost)
		_, isAnte := cost.(*anteCost)
		_, isSource := cost.(*sourceCardCost)
		_, isExile := cost.(*matchingPermanentExileCost)
		_, isDiscard := cost.(*discardCost)
		if random || isSacrifice || isPluralSacrifice || isPlacement || isAnte || isSource || isExile || isDiscard {
			// Stale choices must fail before other resources are spent.
			preflight := *tx
			preflight.game = cloneGameForCostPayment(tx.game)
			if err := preflight.commit(); err != nil {
				return err
			}
			break
		}
	}
	return tx.commit()
}

func (tx *costPaymentTransaction) commit() (commitErr error) {
	if !tx.game.CanLoseLife(tx.controller) {
		if tx.lifeLoss > 0 {
			return fmt.Errorf("cannot pay life")
		}
		for _, cost := range tx.costs {
			if LifePaymentAmount(cost) > 0 {
				return fmt.Errorf("cannot pay life")
			}
		}
	}
	tx.paidXMana = [AnyColor + 1]int{}
	tx.paidMana = [AnyColor + 1]int{}
	tx.paidNumbers = nil
	tx.paidCards = nil
	tx.revealedHandCards = nil
	tx.paidCardSets = nil
	tx.paidPermanents = nil
	tx.paidTaps = nil
	defer func() {
		if commitErr != nil {
			tx.paidXMana = [AnyColor + 1]int{}
			tx.paidMana = [AnyColor + 1]int{}
			tx.paidNumbers = nil
			tx.paidCards = nil
			tx.revealedHandCards = nil
			tx.paidCardSets = nil
			tx.paidPermanents = nil
			tx.paidTaps = nil
		}
	}()
	player := tx.game.GetPlayer(tx.controller)
	if player == nil {
		return ErrPlayerNotFound
	}

	if tx.manaSolution != nil {
		budget := player.Life()
		hasLifeMana := false
		for _, action := range tx.manaSolution.SourcesToTap {
			if action.PlayerManaAbilityID == uuid.Nil {
				continue
			}
			hasLifeMana = true
			found := false
			for _, ability := range tx.game.playerManaAbilities {
				if ability.id == action.PlayerManaAbilityID && ability.player == tx.controller {
					if budget < ability.life {
						return fmt.Errorf("cannot pay total life cost")
					}
					budget -= ability.life
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("planned player mana permission is absent or expired")
			}
		}
		if hasLifeMana {
			if tx.lifeLoss > 0 && budget < tx.lifeLoss {
				return fmt.Errorf("cannot pay total life cost")
			}
			budget -= tx.lifeLoss
			for _, cost := range tx.costs {
				if life, ok := cost.(*lifePayCost); ok && life.amount > 0 {
					if budget < life.amount {
						return fmt.Errorf("cannot pay total life cost")
					}
					budget -= life.amount
				}
			}
		}
	}
	millTotal := 0
	counterTotals := make(map[CounterType]int)
	for _, cost := range tx.costs {
		if reveal, ok := cost.(*revealFromHandCost); ok && !reveal.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen reveal")
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && (exile.capture || exile.captureCard) && !exile.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen graveyard exile")
		}
		if exile, ok := cost.(*exileFromGraveyardCost); ok && !exile.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen graveyard exile")
		}
		if either, ok := cost.(*eitherCost); ok && either.locked {
			if _, exile := either.options[either.selected].(*exileFromGraveyardCost); exile && !either.CanPay(tx.sourceID, tx.controller, tx.game) {
				return fmt.Errorf("cannot pay chosen graveyard exile")
			}
		}
		var counterType CounterType
		var counterAmount int
		switch counter := cost.(type) {
		case *removeCountersCost:
			if !counter.CanPay(tx.sourceID, tx.controller, tx.game) {
				return fmt.Errorf("cannot pay locked counter cost")
			}
			counterType, counterAmount = counter.ct, counter.amount
		case *removeAnyCountersCost:
			if !counter.CanPay(tx.sourceID, tx.controller, tx.game) {
				return fmt.Errorf("cannot pay chosen counter cost")
			}
			counterType, counterAmount = counter.ct, counter.count
		default:
			counterAmount = 0
		}
		if counterAmount > 0 {
			available := tx.game.FindPermanent(tx.sourceID).Counters[counterType]
			if counterAmount > available-counterTotals[counterType] {
				return fmt.Errorf("cannot pay total counter cost")
			}
			counterTotals[counterType] += counterAmount
		}
		if placement, ok := cost.(*putCounterOnMatchingCost); ok && !placement.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen counter placement")
		}
		if taps, ok := cost.(*tapNOtherMatchingCost); ok && !taps.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen tap cost")
		}
		if tap, ok := cost.(*exactPermanentTapCost); ok && !tap.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot tap the declared permanent")
		}
		if attached, ok := cost.(*tapAttachedPermanentCost); ok && !attached.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay attached permanent tap cost")
		}
		if discard, ok := cost.(*discardCost); ok && !discard.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay chosen discard")
		}
		if mill, ok := cost.(*millCost); ok && !mill.CanPay(tx.sourceID, tx.controller, tx.game) {
			return fmt.Errorf("cannot pay mill cost")
		}
		if mill, ok := cost.(*millCost); ok {
			millTotal += mill.lockedAmount
		}
	}
	if millTotal > len(player.Library()) {
		return fmt.Errorf("cannot pay mill cost")
	}
	if err := tx.game.mana.ApplyManaSolution(tx.game, tx.controller, tx.manaSolution); err != nil {
		return err
	}
	if tx.resetDrainedBeforeMana {
		player.ManaPool().ResetLastDrained()
	}
	if !tx.mana.IsZero() {
		if tx.manaX > 0 {
			plan, ok := player.ManaPool().paymentPlan(tx.mana, tx.spellContext)
			if !ok {
				return fmt.Errorf("cannot pay locked mana cost")
			}
			fixed := tx.mana
			fixed.Generic -= min(tx.manaX, fixed.Generic)
			subset := NewManaPool()
			subset.spendingPermissions = player.ManaPool().spendingPermissions
			subset.ManaConversions = player.ManaPool().ManaConversions
			for color, amount := range plan {
				subset.Add(Color(color), amount)
			}
			assigned, ok := subset.paymentPlan(fixed, tx.spellContext)
			if !ok {
				return fmt.Errorf("cannot assign fixed mana requirements")
			}
			for color, amount := range plan {
				tx.paidXMana[color] = amount - assigned[color]
			}
		}
		receipt, err := player.ManaPool().payWithReceipt(tx.mana, tx.spellContext)
		tx.paidMana = receipt
		if err != nil {
			return err
		}
	}
	if tx.lifeLoss > 0 {
		tx.game.PlayerLoseLife(player, tx.lifeLoss)
	}
	for _, cost := range tx.costs {
		if outcome, ok := cost.(*optionalOutcomeCost); ok {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			if outcome.paid {
				tx.paidNumbers[outcome.result] = 1
			} else {
				tx.paidNumbers[outcome.result] = 0
			}
			continue
		}
		var paidPower, paidToughness, paidManaValue int
		var paidCardID uuid.UUID
		var paidPermanent capturedPermanent
		var discardedCardID uuid.UUID
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && (exile.capture || exile.captureCard) {
			for _, card := range tx.game.GetPlayer(tx.controller).Graveyard() {
				if card.ID() == exile.selected && tx.game.graveyardVisit(card.ID()) == exile.visit {
					paidManaValue = card.ManaCost().CMC()
					break
				}
			}
		}
		if discard, ok := cost.(*discardCost); ok && discard.captureCard && len(discard.selected) == 1 {
			discardedCardID = discard.selected[0].id
		}
		paidTotalPower := 0
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureSubtype || sacrifice.captureCard || sacrifice.capturePermanent || sacrifice.captureTotalPower) {
			if !sacrifice.CanPay(tx.sourceID, tx.controller, tx.game) || len(sacrifice.selected) != sacrifice.count {
				return fmt.Errorf("cannot pay chosen sacrifice")
			}
			for _, choice := range sacrifice.selected {
				if sacrifice.captureTotalPower {
					paidTotalPower += tx.game.FindPermanent(choice.id).CurrentPower(tx.game)
				}
			}
			chosen := tx.game.FindPermanent(sacrifice.selected[0].id)
			if sacrifice.capturePermanent {
				paidPermanent = capturedPermanent{id: chosen.ID(), incarnation: chosen.incarnationID}
			}
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
		if sacrifice, ok := cost.(*pluralSacrificeCost); ok {
			payment := *sacrifice
			payment.selected = append([]capturedPermanent(nil), sacrifice.selected...)
			payment.results = nil
			if err := payment.Pay(tx.sourceID, tx.controller, tx.game); err != nil {
				return err
			}
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			if tx.paidCardSets == nil {
				tx.paidCardSets = make(map[*CardsBinding][]capturedCard)
			}
			tx.paidNumbers[payment.power] = payment.total
			tx.paidCardSets[payment.cards] = append([]capturedCard(nil), payment.results...)
			if payment.mandatorySource {
				if tx.paidCards == nil {
					tx.paidCards = make(map[*CardBinding]capturedCard)
				}
				tx.paidCards[payment.sourceBinding] = payment.sourceResult
			}
			continue
		}
		tapVisits := tapCostVisits(cost, tx.sourceID, tx.game)
		if source, ok := cost.(*sourceCardCost); ok {
			result, err := source.payCard(tx.sourceID, tx.controller, tx.game)
			if err != nil {
				return err
			}
			if tx.paidCards == nil {
				tx.paidCards = make(map[*CardBinding]capturedCard)
			}
			tx.paidCards[source.result] = result
		} else if err := cost.Pay(tx.sourceID, tx.controller, tx.game); err != nil {
			return err
		}
		for _, visit := range tapVisits {
			if p := tx.game.FindPermanent(visit.id); p != nil && p.incarnationID == visit.incarnation && p.Tapped {
				tx.paidTaps = append(tx.paidTaps, visit)
			}
		}
		if exile, ok := cost.(*matchingPermanentExileCost); ok && len(exile.facts) != 0 {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			for i, fact := range exile.facts {
				tx.paidNumbers[fact.Result] = exile.values[i]
			}
		}
		if mill, ok := cost.(*millCost); ok && mill.capture {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			tx.paidNumbers[mill.paid] = mill.lockedAmount
		}
		exile, _ := cost.(*exileFromGraveyardCost)
		if exile != nil && exile.paid != nil {
			if tx.paidCardSets == nil {
				tx.paidCardSets = make(map[*CardsBinding][]capturedCard)
			}
			for _, selected := range exile.selected {
				if exiled := tx.game.FindExiledCard(selected.id); exiled != nil {
					tx.paidCardSets[exile.paid] = append(tx.paidCardSets[exile.paid], capturedCard{id: selected.id, visit: exiled.Visit, zone: ZoneExile})
				}
			}
		}
		if taps, ok := cost.(*tapNOtherMatchingCost); ok && taps.capture {
			if tx.paidPermanents == nil {
				tx.paidPermanents = make(map[*PermanentBinding]capturedPermanent)
			}
			tx.paidPermanents[taps.paid] = taps.selected[0]
		}
		if reveal, ok := cost.(*revealFromHandCost); ok {
			identity := capturedCard{id: reveal.selected, visit: reveal.visit, zone: ZoneHand}
			tx.revealedHandCards = append(tx.revealedHandCards, identity)
			if reveal.capture {
				if tx.paidCards == nil {
					tx.paidCards = make(map[*CardBinding]capturedCard)
				}
				tx.paidCards[reveal.paidCard] = identity
			}
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && exile.capture {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			tx.paidNumbers[exile.paid] = paidManaValue
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && exile.captureCard {
			if exiled := tx.game.FindExiledCard(exile.selected); exiled != nil {
				if tx.paidCards == nil {
					tx.paidCards = make(map[*CardBinding]capturedCard)
				}
				tx.paidCards[exile.paidCard] = capturedCard{id: exile.selected, visit: exiled.Visit, zone: ZoneExile}
			}
		}
		if discard, ok := cost.(*discardCost); ok && len(discard.facts) > 0 {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			for i, fact := range discard.facts {
				tx.paidNumbers[fact.Result] = discard.factValues[i]
			}
		}
		if discard, ok := cost.(*discardCost); ok && discard.captureCards {
			if tx.paidCardSets == nil {
				tx.paidCardSets = make(map[*CardsBinding][]capturedCard)
			}
			tx.paidCardSets[discard.paidCards] = append([]capturedCard(nil), discard.results...)
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
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && sacrifice.capturePermanent {
			if tx.paidPermanents == nil {
				tx.paidPermanents = make(map[*PermanentBinding]capturedPermanent)
			}
			tx.paidPermanents[sacrifice.paidPermanent] = paidPermanent
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
		if counter, ok := cost.(*removeCountersCost); ok && counter.capture {
			if tx.paidNumbers == nil {
				tx.paidNumbers = make(map[*NumberBinding]int)
			}
			tx.paidNumbers[counter.paid] = counter.amount
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureSubtype || sacrifice.captureTotalPower) {
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
			if sacrifice.captureSubtype {
				tx.paidNumbers[sacrifice.paidSubtype] = sacrifice.sacrificedSubtype
			}
			if sacrifice.captureTotalPower {
				tx.paidNumbers[sacrifice.paidTotalPower] = paidTotalPower
			}
		}
	}
	for sourceID, paid := range tx.optionalOutcomes {
		tx.game.setOptionalCostPaid(sourceID, paid)
	}
	return nil
}

type actionPaymentSpec struct {
	CaptureManaX             bool
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
	costs, err := lockPermanentManaCosts(spec.Costs, spec.SourceID, spec.Controller, g)
	if err != nil {
		return nil, err
	}
	validation := cloneGameForCostPayment(g)
	locked, outcomes, err := lockPaymentCostsWithManaPolicy(costs, spec.SourceID, spec.Controller, spec.Targets, spec.XValue, validation, player, spec.ApplyActivationReduction)
	if err != nil {
		return nil, err
	}

	var mana ManaCost
	var manaX int
	var nonMana []Cost
	for _, cost := range locked {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			component := payment.MC
			if component.HasX {
				manaX += spec.XValue * max(1, component.XCount)
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

	if !spec.CaptureManaX {
		manaX = 0
	}
	if spec.ApplyActivationReduction {
		mana = g.modifyActivationManaCost(spec.SourceID, mana)
	}

	return g.prepareCostPaymentTransaction(costPaymentSpec{
		ManaX:            manaX,
		Controller:       spec.Controller,
		SourceID:         spec.SourceID,
		Mana:             mana,
		Costs:            nonMana,
		OptionalOutcomes: outcomes,
		Hint:             spec.Hint,
	}, validation)
}
