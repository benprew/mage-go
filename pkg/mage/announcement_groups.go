package mage

import (
	"fmt"
	"slices"
	"uuid"
)

// SpellChoices supplies explicit target groups and an optional modal choice.
type SpellChoices struct {
	TargetGroups [][]uuid.UUID
	Mode         *int
	XValue       int
}

// CastSpellWithChoices casts the specified card with explicit announcement choices.
func (g *Game) CastSpellWithChoices(playerID, cardID uuid.UUID, choices SpellChoices) error {
	player := g.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}
	for _, card := range player.Hand() {
		if card.ID() == cardID {
			return g.castSpellCard(playerID, card, flattenTargetGroups(choices.TargetGroups), choices.TargetGroups, choices.Mode, choices.XValue)
		}
	}
	return ErrCardNotInHand
}

func flattenTargetGroups(groups [][]uuid.UUID) []uuid.UUID {
	var targets []uuid.UUID
	for _, group := range groups {
		targets = append(targets, group...)
	}
	return targets
}
func expandedTargetGroups(specs []Target, groups [][]uuid.UUID) []Target {
	var expanded []Target
	for i, group := range groups {
		if i >= len(specs) {
			break
		}
		for range group {
			expanded = append(expanded, specs[i])
		}
	}
	return expanded
}
func (g *Game) validateTargetGroups(controller uuid.UUID, source *Card, specs []Target, groups [][]uuid.UUID, x int) error {
	if len(specs) != len(groups) {
		return fmt.Errorf("target group count does not match declarations")
	}
	for i, spec := range specs {
		if err := g.validateActionTargets(controller, source, []Target{spec}, groups[i], x, "target group"); err != nil {
			return fmt.Errorf("group %d: %w", i, err)
		}
	}
	return nil
}
func (g *Game) promptTargetGroups(controller uuid.UUID, source *Card, specs []Target, x int) [][]uuid.UUID {
	groups := make([][]uuid.UUID, len(specs))
	for i, spec := range specs {
		groups[i] = g.promptTargetsForList(controller, source, []Target{spec}, x)
	}
	return groups
}
func (g *Game) triggerTargetGroups(trigger *pendingTrigger, specs []Target) [][]uuid.UUID {
	groups := make([][]uuid.UUID, len(specs))
	for i, spec := range specs {
		groups[i] = g.chooseTriggerTargets(trigger, []Target{spec})
	}
	return groups
}
func chooseActionMode(player Player, action *ActionDefinition, reason string, g *Game) int {
	choice := 0
	if chooser, ok := player.(EffectModeChooser); ok {
		choice = chooser.ChooseModeWithEffects(action.modes, reason, g)
	} else if player != nil {
		choice = player.ChooseMode(modalLabels(action), reason)
	}
	return min(max(choice, 0), len(action.modes)-1)
}
func (g *Game) acquireTargetGroups(controller uuid.UUID, card *Card, specs []Target, groups [][]uuid.UUID, x int) [][]uuid.UUID {
	result := cloneTargetGroups(groups)
	for i, spec := range specs {
		if i >= len(result) {
			break
		}
		if random, ok := spec.(*randomTarget); ok {
			result[i] = g.chooseRandomTargets(controller, card, random, x)
		}
		if isOpponentChosenTarget(spec) {
			result[i] = g.acquireOpponentChosenTargets(controller, card, []Target{spec}, slices.Clone(result[i]), x)
		}
	}
	return result
}

func (g *Game) prepareSpellAnnouncement(opts *castStackObjectOptions) error {
	if opts.XValue < 0 {
		return fmt.Errorf("x cannot be negative")
	}
	card, controller, x := opts.Card, opts.Controller, opts.XValue
	specs := card.CastTargets()
	if action, ok := getModalSpellAbility(card); ok {
		if opts.Mode == nil {
			choice := chooseActionMode(g.GetPlayer(controller), action, card.Name(), g)
			opts.Mode = &choice
		}
		if *opts.Mode < 0 || *opts.Mode >= len(action.modes) {
			return fmt.Errorf("invalid modal choice")
		}
		specs = action.modes[*opts.Mode].Targets
	}

	if opts.TargetGroups != nil {
		opts.TargetGroups = g.acquireTargetGroups(controller, card, specs, opts.TargetGroups, x)
		opts.Targets = flattenTargetGroups(opts.TargetGroups)
	} else {
		opts.Targets = g.acquireRandomTargets(controller, card, specs, opts.Targets, x)
		opts.Targets = g.acquireOpponentChosenTargets(controller, card, specs, opts.Targets, x)
		if len(opts.Targets) == 0 && opts.Mode != nil && len(specs) > 0 {
			opts.TargetGroups = g.promptTargetGroups(controller, card, specs, x)
			opts.Targets = flattenTargetGroups(opts.TargetGroups)
		} else {
			opts.TargetGroups = groupTargets(specs, opts.Targets, x)
			if (len(specs) > 0 || opts.Mode != nil) && len(flattenTargetGroups(opts.TargetGroups)) != len(opts.Targets) {
				return fmt.Errorf("too many targets for %s", card.Name())
			}
		}
	}
	return g.validateTargetGroups(controller, card, specs, opts.TargetGroups, x)
}

// ActivateAbilityWithTargetGroups activates an ability with explicit target groups.
func (g *Game) ActivateAbilityWithTargetGroups(playerID, permanentID uuid.UUID, abilityIndex int, groups [][]uuid.UUID) error {
	return g.activateAbilityByIndex(playerID, permanentID, abilityIndex, flattenTargetGroups(groups), groups)
}
