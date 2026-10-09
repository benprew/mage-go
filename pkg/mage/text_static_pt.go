package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithPTTextRevision permits checked replacement of a printed static P/T rule.
// All effects must modify P/T while the source is on the battlefield.
func (a *StaticAbilityHolder) WithPTTextRevision() *StaticAbilityHolder {
	a.ptTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsPTTextRevision() bool {
	if len(a.Effects) == 0 {
		return false
	}
	for _, effect := range a.Effects {
		if nilDefinition(effect) || effect.GetLayer() != core.LayerPT || getEffectSublayer(effect) != core.SublayerPT_Modify || effect.GetDuration() != core.WhileOnBattlefield {
			return false
		}
	}
	return true
}

func (g *Game) printedStaticRegistration(effect ContinuousEffect, source *Permanent, id uuid.UUID) bool {
	registered, ok := effect.(*printedAbilityContinuousEffect)
	return ok && registered.declaration != nil && registered.declaration.AbilityID() == id && registered.SourceID() == source.ID() && registered.incarnation == source.incarnationID
}

func (g *Game) registeredPrintedStaticAbility(source *Permanent, ability Ability) Ability {
	for _, effect := range g.effects.effects {
		if g.printedStaticRegistration(effect, source, ability.AbilityID()) {
			return effect.(*printedAbilityContinuousEffect).declaration
		}
	}
	return ability
}

func printedStaticEffect(effect ContinuousEffect, source *Permanent, declaration *StaticAbilityHolder) ContinuousEffect {
	registered := &printedAbilityContinuousEffect{ContinuousEffect: effect}
	if declaration.ptTextRevision || declaration.abilityTextRevision || declaration.grantTextRevision || declaration.animationTextRevision || declaration.landTypeTextRevision {
		registered.declaration = declaration
		registered.incarnation = source.incarnationID
	}
	return registered
}

// RevisePrintedStaticPTAbility replaces one present printed static P/T rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticPTAbility(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.ptTextRevision && a.supportsPTTextRevision()
	}, changes...)
}

func (g *Game) revisePrintedStaticAbility(source *Permanent, abilityID uuid.UUID, eligible func(*StaticAbilityHolder) bool, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text static revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text static revision requires the original face-up source")
	}
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text static revision requires a retained definition")
	}
	key, ok := base.abilityKeys[abilityID]
	if !ok {
		return fmt.Errorf("text static revision requires a printed declaration")
	}
	index := -1
	var active *StaticAbilityHolder
	for i, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		active, ok = ability.(*StaticAbilityHolder)
		if !ok || !eligible(active) {
			return fmt.Errorf("text static revision requires an eligible printed static rule")
		}
		index = i
		break
	}
	if index < 0 {
		return fmt.Errorf("text static revision requires an active declaration")
	}
	positions := make([]int, 0, len(active.Effects))
	for i, effect := range g.effects.effects {
		if g.printedStaticRegistration(effect, current, abilityID) {
			positions = append(positions, i)
		}
	}
	if len(positions) != len(active.Effects) {
		return fmt.Errorf("text static revision requires complete registrations")
	}
	definition := current.Card
	if active.textRevisionDefinition != nil {
		definition = active.textRevisionDefinition
	}
	declarations, err := definitionAbilitiesByKey(definition.(*BaseCard), true)
	if err != nil {
		return err
	}
	original, ok := declarations[key]
	if !ok {
		return fmt.Errorf("text static revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text static revision lost its replacement")
	}
	fresh, ok := replacement.(*StaticAbilityHolder)
	if !ok || !eligible(fresh) || len(fresh.Effects) != len(active.Effects) {
		return fmt.Errorf("text static revision changed the static contract")
	}
	if active.landTypeTextRevision && active.landTypeTextRevisionContext() != fresh.landTypeTextRevisionContext() {
		return fmt.Errorf("text static revision changed the land-type context")
	}
	for i, effect := range active.Effects {
		next := fresh.Effects[i]
		if effect.GetLayer() != next.GetLayer() || getEffectSublayer(effect) != getEffectSublayer(next) || effect.GetDuration() != next.GetDuration() {
			return fmt.Errorf("text static revision changed an effect layer or duration")
		}
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.textRevisionDefinition = revision.Definition()
	registrations := slices.Clone(g.effects.effects)
	for i, position := range positions {
		effect := g.bindCharacteristicEffect(fresh.Effects[i], current)
		g.setEffectSource(effect, current.ID())
		registrations[position] = printedStaticEffect(effect, current, &rebuilt)
	}
	mutable := g.MutablePermanent(current.ID())
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	mutable.RuntimeAbilities[index] = &rebuilt
	g.effects.effects = registrations
	g.ApplyContinuousEffects()
	return nil
}
