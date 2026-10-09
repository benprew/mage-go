package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// CR 706 — Copying spells. A copy of a spell is created on the stack as a
// brand-new StackObject with the same characteristics as the original (same
// card, same effects, same X value, same chosen mode and same divided-damage
// distribution by default). The controller of the copy is named when the
// copy is created. Targets are copied by default; if the controller of the
// copy is allowed to choose new targets, each target the original spell has
// is reconsidered, and the controller may keep it or pick a legal new one.
//
// A permanent spell copy enters as a token under CR 707.10f.
// Other spell copies cease to exist after resolution or countering.

// CopySpellOnStack copies a spell currently on the stack (CR 706). The
// originalSourceID identifies the stack object by its source card.
// controller controls the copy. If mayChooseNewTargets is true, the copy
// can choose new legal targets from its casting declarations (CR 707.10c).
// Otherwise, the copy keeps the original targets.
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
	var copiedCard Card
	if original.Card != nil && !original.IsAbility {
		copiedCard = original.Card.Copy()
		copiedCard.SetOwner(controller)
	}

	cp := &StackObject{
		ID:                     uuid.New(),
		Card:                   copiedCard,
		Controller:             controller,
		IsAbility:              original.IsAbility,
		activatedAbility:       original.activatedAbility,
		abilitySourceVisit:     original.abilitySourceVisit,
		XValue:                 original.XValue,
		ModeChoice:             original.ModeChoice,
		EventAmount:            original.EventAmount,
		IsCopy:                 true,
		CastZone:               original.CastZone,
		CastContext:            original.CastContext.clone(),
		TargetSource:           copiedCard,
		sourceIncarnation:      original.sourceIncarnation,
		capturedNumbers:        maps.Clone(original.capturedNumbers),
		capturedPlayers:        maps.Clone(original.capturedPlayers),
		capturedCards:          maps.Clone(original.capturedCards),
		capturedCardSets:       cloneCardsBindings(original.capturedCardSets),
		capturedPermanents:     maps.Clone(original.capturedPermanents),
		EventSourceID:          original.EventSourceID,
		targetIncarnations:     maps.Clone(original.targetIncarnations),
		revealedHandCards:      slices.Clone(original.revealedHandCards),
		targetSpellSources:     maps.Clone(original.targetSpellSources),
		targetGraveyardSources: maps.Clone(original.targetGraveyardSources),
		targetHandCards:        maps.Clone(original.targetHandCards),
		targetExileCards:       maps.Clone(original.targetExileCards),
		targetAnteCards:        maps.Clone(original.targetAnteCards),
		boundSpellSources:      maps.Clone(original.boundSpellSources),
	}
	if copiedCard == nil {
		cp.TargetSource = original.TargetSource
		if cp.TargetSource == nil {
			cp.TargetSource = original.Card
		}
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
	cp.TargetSpecs = append([]Target(nil), original.TargetSpecs...)

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

	if mayChooseNewTargets {
		if cp.IsAbility {
			declarations := slices.Compact(slices.Clone(cp.TargetSpecs))
			cp.Targets = g.retargetCopyGroups(cp, g.resolutionTargetSource(cp), declarations)
		} else if copiedCard != nil {
			g.repromptTargetsForCopy(cp, g.resolutionTargetSource(cp))
		}
		if cp.targetIncarnations == nil {
			cp.targetIncarnations = make(map[uuid.UUID]uuid.UUID)
		}
		for i, id := range cp.Targets {
			if i < len(original.Targets) && original.Targets[i] == id {
				continue
			}
			delete(cp.targetIncarnations, id)
			delete(cp.targetSpellSources, id)
			delete(cp.targetGraveyardSources, id)
			delete(cp.targetHandCards, id)
			delete(cp.targetExileCards, id)
			delete(cp.targetAnteCards, id)
			if p := g.FindPermanent(id); p != nil {
				cp.targetIncarnations[id] = p.incarnationID
			} else if visit := g.cardTargetVisit(id); visit != uuid.Nil {
				cp.targetIncarnations[id] = visit
			}
		}
	}

	if mayChooseNewTargets {
		remap := func(values map[uuid.UUID]int) map[uuid.UUID]int {
			if len(values) == 0 {
				return nil
			}
			result := make(map[uuid.UUID]int, len(values))
			for i, old := range original.Targets {
				if i < len(cp.Targets) {
					result[cp.Targets[i]] += values[old]
				}
			}
			return result
		}
		cp.DamageDistribution = remap(original.DamageDistribution)
		cp.CounterDistribution = remap(original.CounterDistribution)
	}

	g.pushStack(cp)
	g.fireBecomesTargetEvents(cp, cp.IsAbility)
	return cp
}

// repromptTargetsForCopy walks the spell's declared Target requirements and
// asks the controller of the copy to choose new targets per CR 706.10c.
// Modal spells reprompt only the chosen mode's targets.
func (g *Game) repromptTargetsForCopy(copyObj *StackObject, card Card) {
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
		newTargets := g.retargetCopyGroups(copyObj, card, mode.Targets)
		if copyObj.ModalTargets == nil {
			copyObj.ModalTargets = make([][]uuid.UUID, len(ms.Modes()))
		}
		copyObj.ModalTargets[modeIdx] = newTargets
		copyObj.Targets = newTargets
		return
	}

	if declarations := card.CastTargets(); len(declarations) > 0 {
		copyObj.Targets = g.retargetCopyGroups(copyObj, card, declarations)
	}
}

// retargetCopyGroups preserves each copied group's count and original choices on cancellation.
func (g *Game) retargetCopyGroups(obj *StackObject, card Card, declarations []Target) []uuid.UUID {
	specs := obj.TargetSpecs
	if len(specs) != len(obj.Targets) {
		specs = expandTargetSpecs(declarations, obj.Targets, obj.XValue)
	}
	out := slices.Clone(obj.Targets)
	offset := 0
	for _, spec := range declarations {
		count := 0
		for offset+count < len(specs) && specs[offset+count] == spec {
			count++
		}
		if count == 0 {
			continue
		}
		old := obj.Targets[offset : offset+count]
		possible := g.targetPlayerQuery(obj.capturedPlayers).possibleTargetsWithX(spec, obj.Controller, card, obj.XValue)
		chosen := g.GetPlayer(obj.Controller).ChooseTargets(possible, count, count, g)
		valid := len(chosen) == count
		seen := make(map[uuid.UUID]bool, count)
		for i, id := range chosen {
			if i >= count || seen[id] || (id != old[i] && !slices.Contains(possible, id)) {
				valid = false
				break
			}
			seen[id] = true
		}
		if valid {
			copy(out[offset:offset+count], chosen)
			for _, id := range chosen {
				visit := g.cardTargetVisit(id)
				if visit == uuid.Nil || !slices.Contains(possible, id) {
					continue
				}
				// A selected card can have a new visit with the same physical ID.
				delete(obj.targetGraveyardSources, id)
				delete(obj.targetHandCards, id)
				delete(obj.targetExileCards, id)
				delete(obj.targetAnteCards, id)
				if obj.targetIncarnations == nil {
					obj.targetIncarnations = make(map[uuid.UUID]uuid.UUID)
				}
				obj.targetIncarnations[id] = visit
			}
		}
		offset += count
	}
	obj.TargetSpecs = specs
	return out
}

// promptTargetsForList walks a flat list of Targets and prompts the
// controller for choices. Used by spell-copy reprompts and by the modal
// cast-target gather.
func (g *Game) promptTargetsForList(controller uuid.UUID, sourceCard Card, targets []Target) []uuid.UUID {
	return g.promptTargetsForListWithX(controller, sourceCard, targets, g.resolution.X())
}

func (g *Game) promptTargetsForListWithX(controller uuid.UUID, sourceCard Card, targets []Target, x int) []uuid.UUID {
	if g.GetPlayer(controller) == nil {
		return nil
	}
	var out []uuid.UUID
	for _, t := range targets {
		minimum, maximum := TargetBounds(t, x)
		if random, ok := randomTargetSpec(t); ok {
			chosen := g.chooseRandomTargets(controller, sourceCard, random, x)
			if len(chosen) == 0 && minimum > 0 {
				out = append(out, uuid.Nil)
			} else {
				out = append(out, chosen...)
			}
			continue
		}
		t.Reset()
		possible := g.possibleTargetsWithX(t, controller, sourceCard, x)
		if len(possible) == 0 {
			if minimum > 0 {
				out = append(out, uuid.Nil)
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
			chosen = possible[:1]
		}
		_ = t.Choose(controller, sourceCard, g, chosen)
		out = append(out, chosen...)
	}
	return out
}
