package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
)

// WithTextRevisionActivationState permits checked state transfer for a printed action.
// A retained definition and declaration group are also required at revision time.
func WithTextRevisionActivationState() ActionOption {
	return func(action *ActionDefinition) { action.textRevisionState = true }
}

// RevisePrintedActivatedAbility replaces one present printed activated action.
// The caller must supply the text layer's order and lifetime separately.
func (g *Game) RevisePrintedActivatedAbility(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if source != nil {
		if p := g.FindPermanent(source.ID()); p != nil && p.layeredActivatedOrigins[abilityID] != nil {
			return fmt.Errorf("direct activation revision cannot replace layered text")
		}
	}
	return g.revisePrintedActivatedAbility(source, abilityID, changes...)
}

func (g *Game) revisePrintedActivatedAbility(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	if source == nil {
		return fmt.Errorf("text activation revision requires a present source")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("text activation revision requires the original face-up source")
	}
	base, ok := current.Card.(*BaseCard)
	if !ok {
		return fmt.Errorf("text activation revision requires a retained definition")
	}
	key, ok := base.abilityKeys[abilityID]
	if !ok {
		return fmt.Errorf("text activation revision requires a printed declaration")
	}
	index := -1
	var active *ActionDefinition
	for i, ability := range current.RuntimeAbilities {
		if ability.AbilityID() != abilityID {
			continue
		}
		active, ok = ability.(*ActionDefinition)
		if !ok || active.kind != ActionActivated || active.staticGrant || !active.textRevisionState {
			return fmt.Errorf("text activation revision requires an eligible printed action")
		}
		index = i
		break
	}
	if index < 0 {
		return fmt.Errorf("text activation revision requires an active declaration")
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
		return fmt.Errorf("text activation revision lost its declaration group")
	}
	revision, err := BuildTextDefinitionRevision(definition, changes...)
	if err != nil {
		return err
	}
	replacement, ok := revision.ReplacementAbility(original.AbilityID())
	if !ok {
		return fmt.Errorf("text activation revision lost its replacement")
	}
	fresh, ok := replacement.(*ActionDefinition)
	if !ok || fresh.kind != ActionActivated || !fresh.textRevisionState || fresh.staticGrant || fresh.activationTextLayer != active.activationTextLayer || (fresh.activationHistory == nil) != (active.activationHistory == nil) {
		return fmt.Errorf("text activation revision changed the activation state contract")
	}
	if active.activationTextLayer && !sameActivatedTextContext(active, fresh) {
		return fmt.Errorf("text activation revision changed the declaration context")
	}
	rebuilt := *fresh
	rebuilt.id = active.id
	rebuilt.source = current.ID()
	rebuilt.controller = current.ControllerID()
	rebuilt.activatedThisTurn = active.activatedThisTurn
	rebuilt.activationsThisTurn = active.activationsThisTurn
	rebuilt.textRevisionDefinition = revision.Definition()
	if fresh.activationHistory != nil {
		fresh.activationHistory.identity = active.activationHistory.identity
	}
	mutable := g.MutablePermanent(current.ID())
	mutable.RuntimeAbilities = slices.Clone(mutable.RuntimeAbilities)
	mutable.RuntimeAbilities[index] = &rebuilt
	retainPrintedActionState(mutable, &rebuilt)
	return nil
}

func retainPrintedActionState(p *Permanent, a *ActionDefinition) {
	if !a.textRevisionState {
		return
	}
	p.revisedPrintedActions = maps.Clone(p.revisedPrintedActions)
	if p.revisedPrintedActions == nil {
		p.revisedPrintedActions = make(map[uuid.UUID]*ActionDefinition)
	}
	p.revisedPrintedActions[a.id] = a
}

func sameActivatedTextContext(a, b *ActionDefinition) bool {
	return a.timing == b.timing && a.stepOnly == b.stepOnly && a.stepMask == b.stepMask &&
		a.limits == b.limits && a.permission == b.permission && a.groupedTargets == b.groupedTargets &&
		len(a.costs) == len(b.costs) && len(a.targets) == len(b.targets) && len(a.modes) == len(b.modes) &&
		len(a.activationConds) == len(b.activationConds) && len(a.dslRestrictions) == len(b.dslRestrictions) &&
		len(a.dslCaptures) == len(b.dslCaptures) && (a.activationManaX == nil) == (b.activationManaX == nil)
}
