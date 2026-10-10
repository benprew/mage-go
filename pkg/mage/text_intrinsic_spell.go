package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithIntrinsicCreatureSpellTextContinuation permits intrinsic rules at entry.
// Copy entry and other permanent rules require separate contracts.
func WithIntrinsicCreatureSpellTextContinuation() CardOption {
	return func(card *BaseCard) { card.intrinsicSpellText++ }
}

func intrinsicSpellTextContext(card Card) error {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.intrinsicSpellText != 1 || !slices.Equal(base.types, []core.CardType{core.TypeCreature}) ||
		len(base.modes) != 0 || len(base.additionalCosts) != 0 || len(base.alternateCosts) != 0 || len(base.castTargets) != 0 || base.landAuraSpellText != 0 {
		return fmt.Errorf("intrinsic spell text requires one marked creature without casting costs or targets")
	}
	landwalk := make(map[core.Attr]int)
	for _, ability := range base.abilities {
		if nilDefinition(ability) {
			return fmt.Errorf("intrinsic spell text requires non-nil declarations")
		}
		switch rule := ability.(type) {
		case *ProtectionAbility:
			if !rule.colorTextRevision || !rule.textLayerEligible || !rule.supportsColorTextRevision() {
				return fmt.Errorf("intrinsic spell text requires marked color protection")
			}
		case *PrintedBasicLandwalkAbility:
			if !rule.textLayerEligible || !isBasicLandwalk(rule.keyword) {
				return fmt.Errorf("intrinsic spell text requires marked basic landwalk")
			}
			landwalk[rule.keyword]++
		default:
			return fmt.Errorf("intrinsic spell text has an unsupported declaration")
		}
	}
	if len(base.abilities) == 0 {
		return fmt.Errorf("intrinsic spell text requires protection or basic landwalk")
	}
	for kw, count := range base.attrSeeds {
		if isBasicLandwalk(kw) {
			if count != landwalk[kw] {
				return fmt.Errorf("intrinsic spell text requires separate landwalk contributions")
			}
			delete(landwalk, kw)
			continue
		}
		if count < 1 || !slices.Contains([]core.Attr{core.Flying, core.FirstStrike, core.Trample, core.Vigilance, core.Haste, core.Reach, core.Banding, core.Defender, core.Fear, core.Deathtouch, core.Lifelink, core.Shroud, core.Hexproof, core.Menace, core.Indestructible}, kw) {
			return fmt.Errorf("intrinsic spell text has an unsupported keyword")
		}
	}
	if len(landwalk) != 0 {
		return fmt.Errorf("intrinsic spell text lost a landwalk contribution")
	}
	return nil
}

func sameIntrinsicCreatureSpellContext(old, fresh *BaseCard, revision *TextDefinitionRevision) bool {
	if !slices.Equal(old.subTypes, fresh.subTypes) {
		return false
	}
	oldAttrs, newAttrs := maps.Clone(old.attrSeeds), maps.Clone(fresh.attrSeeds)
	for kw := range oldAttrs {
		if isBasicLandwalk(kw) {
			delete(oldAttrs, kw)
		}
	}
	for kw := range newAttrs {
		if isBasicLandwalk(kw) {
			delete(newAttrs, kw)
		}
	}
	if !maps.Equal(oldAttrs, newAttrs) {
		return false
	}
	for _, ability := range old.abilities {
		replacement, ok := revision.ReplacementAbility(ability.AbilityID())
		if !ok {
			return false
		}
		switch rule := ability.(type) {
		case *ProtectionAbility:
			next, ok := replacement.(*ProtectionAbility)
			if !ok || len(rule.FromColors) != len(next.FromColors) {
				return false
			}
		case *PrintedBasicLandwalkAbility:
			if _, ok := replacement.(*PrintedBasicLandwalkAbility); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (g *Game) reviseIntrinsicCreatureSpellText(obj *StackObject, changes []TextChange) error {
	if err := intrinsicSpellTextContext(obj.spellTextDefinition); err != nil {
		return err
	}
	if len(obj.Effects) != 0 || len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 || len(obj.capturedNumbers) != 0 ||
		len(obj.capturedPlayers) != 0 || len(obj.capturedPermanents) != 0 || len(obj.capturedCards) != 0 || len(obj.capturedCardSets) != 0 {
		return fmt.Errorf("intrinsic spell text requires no announced targets or casting results")
	}
	revision, err := BuildTextDefinitionRevision(obj.spellTextDefinition, changes...)
	if err != nil {
		return err
	}
	fresh := revision.Definition()
	if err := intrinsicSpellTextContext(fresh); err != nil {
		return err
	}
	if !sameIntrinsicCreatureSpellContext(obj.spellTextDefinition.(*BaseCard), fresh.(*BaseCard), revision) {
		return fmt.Errorf("intrinsic spell text changed its declaration context")
	}
	rebuilt := cloneStackObject(obj)
	rebuilt.spellTextDefinition = fresh
	for i, current := range g.stack.objects {
		if current.ID == obj.ID {
			g.stack.objects[i] = rebuilt
			return nil
		}
	}
	return fmt.Errorf("intrinsic spell text lost its stack visit")
}

func buildIntrinsicCreatureSpellEntry(obj *StackObject) (Card, error) {
	if err := intrinsicSpellTextContext(obj.spellTextDefinition); err != nil {
		return nil, err
	}
	fresh, err := BuildTextChangedDefinition(obj.spellTextDefinition)
	if err != nil {
		return nil, err
	}
	base := fresh.(*BaseCard)
	base.id, base.owner, base.isToken = obj.Card.ID(), obj.Card.Owner(), obj.Card.IsToken()
	base.departureCard = obj.Card
	characteristics := printedCopiableCharacteristics(base)
	base.textEntryValues = &characteristics
	return base, nil
}
