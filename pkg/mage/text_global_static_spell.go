package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithGlobalStaticSpellTextContinuation permits one checked static rule at entry.
// Casting costs, targets and other rules require separate contracts.
func WithGlobalStaticSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.globalStaticSpellText++ }
}

func globalStaticSpellTextContext(card Card) (staticTextContract, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.globalStaticSpellText != 1 || !isEntryCopyPermanent(base) || base.HasType(core.TypeLand) ||
		base.HasType(core.TypeInstant) || base.HasType(core.TypeSorcery) || base.HasSubType("Aura") || len(base.attrSeeds) != 0 ||
		len(base.abilities) != 1 || len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 ||
		base.intrinsicSpellText != 0 || base.landAuraSpellText != 0 || base.copyEntrySpellText != 0 {
		return 0, fmt.Errorf("global static spell text requires one marked permanent rule without casting costs or targets")
	}
	rule, ok := base.abilities[0].(*StaticAbilityHolder)
	if ok && rule != nil && rule.grantTextLayer && rule.grantTextRevision && rule.supportsGrantTextRevision() {
		return staticGrantTextContract, nil
	}
	if ok && rule != nil && len(rule.Effects) == 1 && rule.ptTextLayer && rule.ptTextRevision && rule.supportsPTTextRevision() {
		kind := rule.ptTextRevisionContext().kind
		if kind == controllerBasicLandSourcePT || kind == opponentBasicLandSourcePT {
			return staticPTTextContract, nil
		}
	}
	return globalStaticSpellRuleContext(base.abilities[0])
}

func globalStaticSpellRuleContext(ability Ability) (staticTextContract, error) {
	rule, ok := ability.(*StaticAbilityHolder)
	if !ok || rule == nil || len(rule.Effects) != 1 {
		return 0, fmt.Errorf("global static spell text requires one static effect")
	}
	if rule.landTypeTextLayer && rule.landTypeTextRevision && rule.landTypeTextRevisionContext() == globalLandTypeRevision {
		return staticLandTypeTextContract, nil
	}
	if rule.ptTextLayer && rule.ptTextRevision && rule.supportsPTTextRevision() && rule.ptTextRevisionContext() == (ptRevisionContext{}) {
		return staticPTTextContract, nil
	}
	return 0, fmt.Errorf("global static spell text requires a marked global P/T or basic land rule")
}

func (g *Game) reviseGlobalStaticSpellText(obj *StackObject, changes []TextChange) error {
	oldContext, err := globalStaticSpellTextContext(obj.spellTextDefinition)
	if err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("global static spell text requires no announced targets, bodies or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh := revision.Definition()
	newContext, err := globalStaticSpellTextContext(fresh)
	if err != nil {
		return err
	}
	oldBase, newBase := obj.spellTextDefinition.(*BaseCard), fresh.(*BaseCard)
	replacement, paired := revision.ReplacementAbility(oldBase.abilities[0].AbilityID())
	samePT := true
	if oldContext == staticPTTextContract && newContext == staticPTTextContract {
		samePT = oldBase.abilities[0].(*StaticAbilityHolder).ptTextRevisionContext() == newBase.abilities[0].(*StaticAbilityHolder).ptTextRevisionContext()
	}
	sameGrant := true
	if oldContext == staticGrantTextContract && newContext == staticGrantTextContract {
		sameGrant = sameGlobalGrantSpellEffects(oldBase.abilities[0].(*StaticAbilityHolder), newBase.abilities[0].(*StaticAbilityHolder))
	}
	if !samePT || !sameGrant || newContext != oldContext || !slices.Equal(oldBase.subTypes, newBase.subTypes) || !paired || replacement != newBase.abilities[0] {
		return fmt.Errorf("global static spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = fresh
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("global static spell text lost its stack visit")
}

func sameGlobalGrantSpellEffects(old, fresh *StaticAbilityHolder) bool {
	if len(old.Effects) != len(fresh.Effects) {
		return false
	}
	for i, effect := range old.Effects {
		next := fresh.Effects[i]
		if effect.GetLayer() != next.GetLayer() || getEffectSublayer(effect) != getEffectSublayer(next) || effect.GetDuration() != next.GetDuration() {
			return false
		}
	}
	return true
}

func buildGlobalStaticSpellEntry(obj *StackObject) (Card, error) {
	if _, err := globalStaticSpellTextContext(obj.spellTextDefinition); err != nil {
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
