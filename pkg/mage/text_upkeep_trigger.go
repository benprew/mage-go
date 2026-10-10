package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type chosenPlayerUpkeepPattern struct{ player *PlayerBinding }

// ChosenPlayerUpkeep saves the player chosen before the source entered.
// The choice stays fixed when the source changes controller.
func ChosenPlayerUpkeep(player *PlayerBinding) EventPattern {
	return chosenPlayerUpkeepPattern{player: player}
}

func (chosenPlayerUpkeepPattern) eventPattern() {}
func (p chosenPlayerUpkeepPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return EachUpkeep.makeStepTrigger(effect).AndConditionData(EventIsChosenPlayerUpkeep{})
}
func (p chosenPlayerUpkeepPattern) inspectDefinition() definitionNode {
	n := (turnStepEventPattern{EachUpkeep, p.player}).inspectDefinition()
	n.Text = "beginning of chosen player's upkeep"
	return n
}
func (p chosenPlayerUpkeepPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return (turnStepEventPattern{EachUpkeep, p.player}).captureEventEffects(event, effects)
}

// WithUpkeepTextRevision permits checked replacement of a printed upkeep rule.
// Other event and source-zone contracts require separate services.
func (t *GenericTriggered) WithUpkeepTextRevision() *GenericTriggered {
	t.upkeepTextRevision = true
	return t
}

type upkeepTextContext struct{ controllerOnly, chosen bool }

func (t *GenericTriggered) upkeepTextContext() (upkeepTextContext, bool) {
	if t == nil || t.eventType != core.EvtUpkeep || t.sourceZone != core.ZoneBattlefield || len(t.zones) != 0 ||
		t.isStateTrigger || t.Optional || t.interveningDefinition != nil || t.stateDefinition != nil || len(t.modes) != 0 || len(t.effects) != 1 {
		return upkeepTextContext{}, false
	}
	switch p := t.eventDefinition.(type) {
	case turnStepEventPattern:
		step, ok := p.step.(stepPattern)
		if !ok || (step != EachUpkeep && step != YourUpkeep) || p.player == nil {
			return upkeepTextContext{}, false
		}
		return upkeepTextContext{controllerOnly: step.controllerOnly}, true
	case chosenPlayerUpkeepPattern:
		return upkeepTextContext{chosen: true}, p.player != nil
	}
	return upkeepTextContext{}, false
}

func (t *GenericTriggered) supportsUpkeepTextRevision() bool {
	_, ok := t.upkeepTextContext()
	return ok
}

// RevisePrintedUpkeepTrigger replaces one present printed upkeep declaration.
// The caller must supply text-layer order, duration and baseline restoration.
func (g *Game) RevisePrintedUpkeepTrigger(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if len(changes) == 0 {
		return fmt.Errorf("upkeep revision requires text changes")
	}
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			for _, ability := range current.RuntimeAbilities {
				if nilDefinition(ability) {
					return fmt.Errorf("upkeep revision requires valid runtime declarations")
				}
			}
		}
	}
	var context upkeepTextContext
	var targets []Target
	checked := false
	return g.revisePrintedTrigger(source, abilityID, func(t *GenericTriggered) bool {
		fresh, ok := t.upkeepTextContext()
		if !ok || !t.upkeepTextRevision {
			return false
		}
		if !checked {
			context, targets, checked = fresh, t.targets, true
		}
		if fresh != context || len(t.targets) != len(targets) {
			return false
		}
		for i, target := range targets {
			if !sameSpellTextTarget(target, t.targets[i]) {
				return false
			}
		}
		return true
	}, changes...)
}
