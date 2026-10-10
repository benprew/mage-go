package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithMixedStaticEventSpellTextContinuation permits paired rules at entry.
// Casting results and other rules require separate contracts.
func WithMixedStaticEventSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.staticEventSpellText++ }
}

func mixedStaticEventSpellTextContext(card Card) (int, staticTextContract, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.staticEventSpellText != 1 || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) ||
		base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.attrSeeds) != 0 ||
		len(base.abilities) != 2 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.manaTapSpellText != 0 || base.spellCastSpellText != 0 || base.globalStaticSpellText != 0 || base.intrinsicSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return 0, 0, fmt.Errorf("mixed static/event spell text requires two marked permanent rules without casting costs or targets")
	}
	for i, rule := range base.abilities {
		context, err := globalStaticSpellRuleContext(rule)
		if err != nil {
			continue
		}
		switch event := base.abilities[1-i].(type) {
		case *ManaBonusAbility:
			if event != nil && layeredManaTapEligible(event) && !event.AttachedOnly && !event.MatchProduced && event.controllerRecipient {
				return i, context, nil
			}
		case *GenericTriggered:
			if _, valid := mixedUpkeepPayment(event); valid {
				return i, context, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("mixed static/event spell text requires one global static rule and one controller bonus or upkeep mana payment")
}

func singleMixedRuleEffect(effect Effect) Effect {
	for {
		sequence, ok := effect.(*sequenceEffect)
		if !ok || sequence == nil || len(sequence.effects) != 1 {
			return effect
		}
		effect = sequence.effects[0]
	}
}

func mixedUpkeepPayment(rule *GenericTriggered) (*ManaCostPayment, bool) {
	if rule == nil || rule.sourceZone != core.ZoneBattlefield || rule.Optional || len(rule.zones) != 0 || len(rule.targets) != 0 || len(rule.modes) != 0 ||
		rule.isStateTrigger || rule.interveningDefinition != nil || rule.stateDefinition != nil || len(rule.effects) != 1 {
		return nil, false
	}
	pattern, ok := rule.eventDefinition.(turnStepEventPattern)
	step, validStep := pattern.step.(stepPattern)
	if !ok || !validStep || step != YourUpkeep || rule.eventType != core.EvtUpkeep {
		return nil, false
	}
	payment, ok := singleMixedRuleEffect(rule.effects[0]).(*unlessPaidEffect)
	if !ok || payment == nil {
		return nil, false
	}
	_, controllerPayer := payment.player.(controllerRef)
	if !controllerPayer || payment.ifPaid || payment.required || payment.twoBranches || payment.otherwise != nil || !payment.costs.present || len(payment.costs.costs) != 1 {
		return nil, false
	}
	mana, ok := payment.costs.costs[0].(*ManaCostPayment)
	if !ok || mana == nil || mana.MC.HasX || mana.MC.XCount != 0 {
		return nil, false
	}
	sacrifice, ok := singleMixedRuleEffect(payment.effect).(*sacrificeEffect)
	if !ok || sacrifice == nil {
		return nil, false
	}
	_, self := sacrifice.subject.(sourceRef)
	if !self || sacrifice.optionCount != 0 || sacrifice.invalidOption {
		return nil, false
	}
	return mana, true
}

func sameMixedEventSpellContext(old, fresh Ability) bool {
	switch first := old.(type) {
	case *ManaBonusAbility:
		return sameManaTapSpellTextContext(first, fresh)
	case *GenericTriggered:
		second, ok := fresh.(*GenericTriggered)
		if !ok {
			return false
		}
		a, validA := mixedUpkeepPayment(first)
		b, validB := mixedUpkeepPayment(second)
		if !validA || !validB || a.MC.String() != b.MC.String() {
			return false
		}
		return (first.eventDefinition.(turnStepEventPattern).player == nil) == (second.eventDefinition.(turnStepEventPattern).player == nil)
	}
	return false
}

func (g *Game) reviseMixedStaticEventSpellText(obj *StackObject, changes []TextChange) error {
	index, context, err := mixedStaticEventSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 || len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("mixed static/event spell text requires no announced targets, bodies or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	newIndex, newContext, err := mixedStaticEventSpellTextContext(revision.Definition())
	if err != nil {
		return err
	}
	old, fresh := obj.spellTextDefinition.(*BaseCard), revision.Definition().(*BaseCard)
	if index != newIndex || context != newContext || !slices.Equal(old.subTypes, fresh.subTypes) || !sameMixedEventSpellContext(old.abilities[1-index], fresh.abilities[1-index]) {
		return fmt.Errorf("mixed static/event spell text changed its declaration context")
	}
	for i, rule := range old.abilities {
		replacement, paired := revision.ReplacementAbility(rule.AbilityID())
		if !paired || replacement != fresh.abilities[i] {
			return fmt.Errorf("mixed static/event spell text lost its paired declarations")
		}
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = revision.Definition()
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("mixed static/event spell text lost its stack visit")
}

func buildMixedStaticEventSpellEntry(obj *StackObject) (Card, error) {
	if _, _, err := mixedStaticEventSpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	fresh, err := BuildTextChangedDefinition(obj.spellTextDefinition)
	if err != nil {
		return nil, err
	}
	base := fresh.(*BaseCard)
	base.id, base.owner, base.isToken = obj.Card.ID(), obj.Card.Owner(), obj.Card.IsToken()
	base.departureCard = obj.Card
	return base, nil
}
