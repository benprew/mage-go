package mage

import (
	"fmt"
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
	if !ok || fresh.kind != ActionActivated || !fresh.textRevisionState || fresh.staticGrant || (fresh.activationHistory == nil) != (active.activationHistory == nil) {
		return fmt.Errorf("text activation revision changed the activation state contract")
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
	return nil
}
