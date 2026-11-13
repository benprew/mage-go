package mage

import (
	"maps"
	"slices"
	"uuid"
)

// CR 706 — Copying spells. A copy of a spell is created on the stack as a
// brand-new StackObject with the same characteristics as the original (same
// card, same effects, same X value, same chosen mode and same divided-damage
// distribution by default). The controller of the copy is named when the
// copy is created. Targets are copied by default; if the controller of the
// copy is allowed to choose new targets, each target the original spell has
// is reconsidered, and the controller may keep it or pick a legal new one.
//
// Per CR 707.10, a copy of a spell ceases to exist as it resolves or is
// countered. ResolveStackObject honors StackObject.IsCopy to skip the
// to-graveyard / to-battlefield steps.

// CopySpellOnStack copies a spell currently on the stack (CR 706). The
// originalSourceID is the SourceID of the StackObject to copy (matching the
// card ID for spells). controller is the player who controls the copy
// (CR 706.10c). If mayChooseNewTargets is true, each Target on the spell's
// SpellAbility is re-prompted on the controller of the copy; otherwise the
// original targets are carried over verbatim (CR 706.10c).
//
// Returns the new StackObject (already pushed on top of the stack), or nil
// if the original spell was not on the stack.
func (g *Game) CopySpellOnStack(originalSourceID, controller uuid.UUID, mayChooseNewTargets bool) *StackObject {
	original := g.stack.FindBySourceID(originalSourceID)
	if original == nil {
		return nil
	}
	return g.copyStackObject(original, controller, mayChooseNewTargets)
}

// CopyStackObjectDirect copies the supplied StackObject (already located by
// the caller). Behavior matches CopySpellOnStack.
func (g *Game) CopyStackObjectDirect(obj *StackObject, controller uuid.UUID, mayChooseNewTargets bool) *StackObject {
	if obj == nil {
		return nil
	}
	return g.copyStackObject(obj, controller, mayChooseNewTargets)
}

func (g *Game) copyStackObject(original *StackObject, controller uuid.UUID, mayChooseNewTargets bool) *StackObject {
	var copiedCard *Card
	if original.Card != nil {
		copiedCard = original.Card.NewCopy()
		copiedCard.SetOwner(controller)
	}

	cp := &StackObject{
		ID:           uuid.New(),
		Card:         copiedCard,
		Controller:   controller,
		IsAbility:    original.IsAbility,
		XValue:       original.XValue,
		ModeChoice:   original.ModeChoice,
		EventAmount:  original.EventAmount,
		IsCopy:       true,
		CastZone:     original.CastZone,
		CastContext:  cloneCastContext(original.CastContext),
		TargetSource: copiedCard,
	}
	if copiedCard != nil {
		cp.SourceID = copiedCard.ID()
	} else {
		cp.SourceID = original.SourceID
	}

	if len(original.Effects) > 0 {
		cp.Effects = make([]Effect, len(original.Effects))
		copy(cp.Effects, original.Effects)
	}

	cp.Targets = append([]uuid.UUID(nil), original.Targets...)
	cp.TargetGroups = cloneTargetGroups(original.TargetGroups)
	cp.TriggerEvent = newGameCloneContext().event(original.TriggerEvent)
	cp.RememberedObjects = slices.Clone(original.RememberedObjects)
	cp.rememberedZones = maps.Clone(original.rememberedZones)
	cp.rememberedIncarnations = maps.Clone(original.rememberedIncarnations)
	cp.TargetSpecs = slices.Clone(original.TargetSpecs)
	cp.TargetZones = maps.Clone(original.TargetZones)
	cp.TargetIncarnations = maps.Clone(original.TargetIncarnations)

	if len(original.ModalTargets) > 0 {
		cp.ModalTargets = make([][]uuid.UUID, len(original.ModalTargets))
		for i, t := range original.ModalTargets {
			cp.ModalTargets[i] = append([]uuid.UUID(nil), t...)
		}
	}

	if len(original.DamageDistribution) > 0 {
		cp.DamageDistribution = make(map[uuid.UUID]int, len(original.DamageDistribution))
		maps.Copy(cp.DamageDistribution, original.DamageDistribution)
	}
	if len(original.CounterDistribution) > 0 {
		cp.CounterDistribution = make(map[uuid.UUID]int, len(original.CounterDistribution))
		maps.Copy(cp.CounterDistribution, original.CounterDistribution)
	}

	if mayChooseNewTargets && copiedCard != nil {
		g.repromptTargetsForCopy(cp, copiedCard)
	}

	g.pushStack(cp)
	g.fireBecomesTargetEvents(cp, cp.IsAbility)
	return cp
}

// repromptTargetsForCopy walks the spell's declared Target requirements and
// asks the controller of the copy to choose new targets per CR 706.10c.
// Modal spells reprompt only the chosen mode's targets.
func (g *Game) repromptTargetsForCopy(copyObj *StackObject, card *Card) {
	controller := g.GetPlayer(copyObj.Controller)
	if controller == nil {
		return
	}

	if ms, ok := getModalSpellAbility(card); ok {
		modeIdx := copyObj.ModeChoice
		if modeIdx < 0 || modeIdx >= len(ms.Modes()) {
			return
		}
		mode := ms.Modes()[modeIdx]
		groups := g.promptTargetGroups(controller.PlayerID(), card, mode.Targets, copyObj.XValue)
		newTargets := flattenTargetGroups(groups)
		copyObj.TargetGroups = groups
		copyObj.TargetSpecs = expandedTargetGroups(mode.Targets, groups)
		if copyObj.ModalTargets == nil {
			copyObj.ModalTargets = make([][]uuid.UUID, len(ms.Modes()))
		}
		copyObj.ModalTargets[modeIdx] = newTargets
		copyObj.Targets = newTargets
		copyObj.TargetZones = nil
		copyObj.TargetIncarnations = nil
		return
	}

	for _, a := range card.abilities {
		sa, ok := a.(*SpellAbility)
		if !ok {
			continue
		}
		if len(sa.Targets()) == 0 {
			continue
		}
		groups := g.promptTargetGroups(controller.PlayerID(), card, sa.Targets(), copyObj.XValue)
		newTargets := flattenTargetGroups(groups)
		copyObj.TargetGroups = groups
		copyObj.TargetSpecs = expandedTargetGroups(sa.Targets(), groups)
		copyObj.Targets = newTargets
		copyObj.TargetZones = nil
		copyObj.TargetIncarnations = nil
		return
	}
}

// promptTargetsForList walks a flat list of Targets and prompts the
// controller for choices. Used by spell-copy reprompts and by the modal
// cast-target gather.
func (g *Game) promptTargetsForList(controller uuid.UUID, sourceCard *Card, targets []Target, x int) []uuid.UUID {
	if g.GetPlayer(controller) == nil {
		return nil
	}
	var out []uuid.UUID
	for _, t := range targets {
		minimum, maximum := TargetBounds(t, x)
		if random, ok := t.(*randomTarget); ok {
			chosen := g.chooseRandomTargets(controller, sourceCard, random, x)
			if len(chosen) == 0 && minimum > 0 {
				out = append(out, uuid.Nil())
			} else {
				out = append(out, chosen...)
			}
			continue
		}

		possible := PossibleTargets(t, controller, sourceCard, g, TargetingContext{X: x, XSpecified: true})
		if len(possible) == 0 {
			if minimum > 0 {
				out = append(out, uuid.Nil())
			}
			continue
		}
		chooser := g.GetPlayer(controller)
		if isOpponentChosenTarget(t) {
			chooser = g.GetOpponent(controller)
		}
		var chosen []uuid.UUID
		if chooser != nil {
			chosen = chooser.ChooseTargets(possible, minimum, maximum, g)
		}
		if len(chosen) == 0 && minimum > 0 {
			chosen = possible[:min(minimum, len(possible))]
		}

		out = append(out, chosen...)
	}
	return out
}
