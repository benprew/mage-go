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
		ID:                uuid.New(),
		Card:              copiedCard,
		Controller:        controller,
		IsAbility:         original.IsAbility,
		XValue:            original.XValue,
		ModeChoice:        original.ModeChoice,
		EventAmount:       original.EventAmount,
		EventSourceID:     original.EventSourceID,
		SourceIncarnation: original.SourceIncarnation,
		IsCopy:            true,
		CastZone:          original.CastZone,
		CastContext:       cloneCastContext(original.CastContext),
		TargetSource:      copiedCard,
		declaredTargets:   original.declaredTargets,
		damageTotal:       original.damageTotal,
		counterTotal:      original.counterTotal,
	}
	if copiedCard != nil {
		cp.SourceID = copiedCard.ID()
	} else {
		cp.SourceID = original.SourceID
		cp.TargetSource = original.TargetSource
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

	if mayChooseNewTargets && copiedCard != nil {
		g.repromptTargetsForCopy(cp, copiedCard)
	}
	cp.DamageDistribution = remapTargetShares(original.DamageDistribution, original.Targets, cp.Targets)
	cp.CounterDistribution = remapTargetShares(original.CounterDistribution, original.Targets, cp.Targets)

	g.pushStack(cp)
	g.fireBecomesTargetEvents(cp, cp.IsAbility)
	return cp
}

// repromptTargetsForCopy preserves the target slots and lets the controller change their targets.
func (g *Game) repromptTargetsForCopy(copyObj *StackObject, card *Card) {
	controller := g.GetPlayer(copyObj.Controller)
	if controller == nil {
		return
	}

	specs := card.CastTargets()
	if ms, ok := getModalSpellAbility(card); ok {
		modeIdx := copyObj.ModeChoice
		if modeIdx < 0 || modeIdx >= len(ms.Modes()) {
			return
		}
		specs = ms.Modes()[modeIdx].Targets
	}
	if len(specs) == 0 {
		return
	}
	groups := copyObj.TargetGroups
	if groups == nil {
		groups = groupTargets(specs, copyObj.Targets, copyObj.XValue)
	}
	if len(groups) != len(specs) || len(flattenTargetGroups(groups)) != len(copyObj.Targets) {
		return
	}
	targeting := TargetingContext{X: copyObj.XValue, XSpecified: true, StackObjectID: copyObj.ID}
	for i, spec := range specs {
		previous := groups[i]
		if len(previous) == 0 {
			continue
		}
		legal := PossibleTargets(spec, controller.PlayerID(), card, g, targeting)
		choices := slices.Clone(legal)
		for _, id := range previous {
			if id != uuid.Nil() && !slices.Contains(choices, id) {
				choices = append(choices, id)
			}
		}
		chooser := controller
		if isOpponentChosenTarget(spec) {
			chooser = g.GetOpponent(controller.PlayerID())
		}
		if chooser == nil || len(choices) < len(previous) {
			continue
		}
		chosen := chooser.ChooseTargets(choices, len(previous), len(previous), g)
		if !validCopyTargets(previous, chosen, legal) {
			continue
		}
		for slot, id := range chosen {
			if id == previous[slot] || copyObj.TargetZones == nil {
				continue
			}
			copyObj.TargetZones[id] = g.zoneOfTarget(id)
			if p := g.FindPermanent(id); p != nil {
				if copyObj.TargetIncarnations == nil {
					copyObj.TargetIncarnations = make(map[uuid.UUID]uuid.UUID)
				}
				copyObj.TargetIncarnations[id] = p.incarnationID
			} else {
				delete(copyObj.TargetIncarnations, id)
			}
		}
		groups[i] = slices.Clone(chosen)
	}
	copyObj.TargetGroups = groups
	copyObj.Targets = flattenTargetGroups(groups)
	copyObj.TargetSpecs = expandedTargetGroups(specs, groups)
	if copyObj.ModeChoice >= 0 && copyObj.ModeChoice < len(copyObj.ModalTargets) {
		copyObj.ModalTargets[copyObj.ModeChoice] = slices.Clone(copyObj.Targets)
	}
}

func validCopyTargets(previous, chosen, legal []uuid.UUID) bool {
	if len(chosen) != len(previous) {
		return false
	}
	for i, id := range chosen {
		if id == uuid.Nil() || slices.Contains(chosen[:i], id) || id != previous[i] && !slices.Contains(legal, id) {
			return false
		}
	}
	return true
}

func remapTargetShares(shares map[uuid.UUID]int, previous, next []uuid.UUID) map[uuid.UUID]int {
	if len(shares) == 0 {
		return maps.Clone(shares)
	}
	result := make(map[uuid.UUID]int, len(shares))
	for i, id := range previous {
		if amount, ok := shares[id]; ok {
			result[next[i]] = amount
		}
	}
	return result
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
