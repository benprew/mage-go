package mage

import (
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ObjectSelector reads object references from the execution context.
type ObjectSelector interface {
	Select(*EffectContext) []uuid.UUID
	Text() string
}

// Selector selects object or player identifiers.
type Selector interface {
	ObjectSelector
	PlayerSelector
}

type objectReference struct {
	kind  string
	index int
}

// TargetGroup selects the declared targets in one group.
func TargetGroup(index int) Selector { return objectReference{kind: "group", index: index} }

// FirstTarget selects the first declared target.
func FirstTarget() Selector { return objectReference{kind: "first"} }

// SourceObject selects the source of the resolving effect.
func SourceObject() ObjectSelector { return objectReference{kind: "source"} }

// EventSource selects the source recorded in the triggering event.
func EventSource() ObjectSelector { return objectReference{kind: "event source"} }

// EventTarget selects the object recorded as the event target.
func EventTarget() ObjectSelector { return objectReference{kind: "event target"} }

// EventPlayer selects the player recorded in the triggering event.
func EventPlayer() PlayerSelector { return objectReference{kind: "event player"} }

// EventAmount reads the amount recorded in the triggering event.
func EventAmount() ValueSource         { return eventSnapshotAmount{} }
func (r objectReference) Text() string { return r.kind }
func (r objectReference) Select(ctx *EffectContext) []uuid.UUID {
	if ctx == nil {
		return nil
	}
	var id uuid.UUID
	switch r.kind {
	case "group":
		if r.index < 0 || r.index >= len(ctx.TargetGroups) {
			return nil
		}
		return slices.Clone(ctx.TargetGroups[r.index])
	case "remembered":
		if r.index < 0 || r.index >= len(ctx.RememberedObjects) {
			return nil
		}
		id = ctx.RememberedObjects[r.index]
	case "first":
		if len(ctx.Targets) > 0 {
			id = ctx.Targets[0]
		}
	case "source":
		id = ctx.SourceID
	case "event source":
		if ctx.TriggerEvent != nil {
			id = ctx.TriggerEvent.SourceID
		}
	case "event target":
		if ctx.TriggerEvent != nil {
			id = ctx.TriggerEvent.TargetID
		}
	case "event player":
		if ctx.TriggerEvent != nil {
			id = ctx.TriggerEvent.PlayerID
		}
	}
	if id == uuid.Nil() {
		return nil
	}
	return []uuid.UUID{id}
}

func cloneTargetGroups(groups [][]uuid.UUID) [][]uuid.UUID {
	if groups == nil {
		return nil
	}
	result := make([][]uuid.UUID, len(groups))
	for i := range groups {
		result[i] = slices.Clone(groups[i])
	}
	return result
}

func groupTargets(specifications []Target, targets []uuid.UUID, x int) [][]uuid.UUID {
	groups := make([][]uuid.UUID, len(specifications))
	offset := 0
	for i, spec := range specifications {
		_, maximum := TargetBounds(spec, x)
		required := 0
		for _, later := range specifications[i+1:] {
			minimum, _ := TargetBounds(later, x)
			required += minimum
		}
		count := min(maximum, max(0, len(targets)-offset-required))
		groups[i] = slices.Clone(targets[offset : offset+count])
		offset += count
	}
	return groups
}

func resolvedTargetGroups(groups [][]uuid.UUID, targets []uuid.UUID) [][]uuid.UUID {
	result := cloneTargetGroups(groups)
	offset := 0
	for i := range result {
		for j := range result[i] {
			if offset < len(targets) {
				result[i][j] = targets[offset]
			}
			offset++
		}
	}
	return result
}

func (ctx *EffectContext) legacyTargets() []uuid.UUID {
	if len(ctx.Targets) > 0 || ctx.declaredTargets {
		return ctx.Targets
	}
	if len(ctx.RememberedObjects) > 0 {
		return ctx.RememberedObjects
	}
	if ctx.TriggerEvent == nil {
		return nil
	}
	event := ctx.TriggerEvent
	switch event.Type {
	case EvtZoneChange:
		if event.ToZone == ZoneBattlefield {
			return referenceIDs(event.SourceID)
		}
		if event.FromZone == ZoneBattlefield {
			return referenceIDs(event.SourceID, event.PlayerID)
		}
	case EvtSpellCast:
		return referenceIDs(event.SourceID, event.PlayerID)
	case EvtDrawStep, EvtCardDrawn, EvtTappedForMana, EvtDiscard, EvtSacrifice:
		return referenceIDs(event.PlayerID)
	case EvtDamageDealt:
		return referenceIDs(event.TargetID)
	case EvtTapped, EvtAbilityActivated, EvtDeclaredAttacker:
		return referenceIDs(event.SourceID)
	case EvtBecomesTarget:
		return referenceIDs(event.TargetID, event.SourceID)
	case EvtDeclaredBlocker:
		return referenceIDs(event.SourceID, event.TargetID)
	}
	return nil
}

func referenceIDs(ids ...uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil() {
			result = append(result, id)
		}
	}
	return result
}

func currentObjectIDs(ctx *EffectContext, selector ObjectSelector) []uuid.UUID {
	if reference, ok := selector.(objectReference); ok && reference.kind == "source" && ctx.SourcePerm != nil {
		current := ctx.Game.FindPermanent(ctx.SourceID)
		if current == nil || current.incarnationID != ctx.SourcePerm.incarnationID {
			return nil
		}
	}
	return selector.Select(ctx)
}

type objectControllerReference struct{ objects ObjectSelector }

// ObjectController selects the current controllers of the selected objects.
func ObjectController(objects ObjectSelector) PlayerSelector {
	return objectControllerReference{objects: objects}
}
func (objectControllerReference) Text() string { return "that object's controller" }
func (s objectControllerReference) Select(ctx *EffectContext) []uuid.UUID {
	var players []uuid.UUID
	for _, id := range s.objects.Select(ctx) {
		if object := ctx.Game.LookupObject(id); object != nil {
			player := object.ViewController()
			if !slices.Contains(players, player) {
				players = append(players, player)
			}
		}
	}
	return players
}

type libraryTopReference struct {
	player PlayerSelector
	amount ValueSource
	mill   bool
}

// LibraryTopObjects selects cards from the top of the selected players' libraries.
func LibraryTopObjects(player PlayerSelector, amount ValueSource) ObjectSelector {
	return libraryTopReference{player: player, amount: amount}
}
func (libraryTopReference) Text() string { return "the top cards of the library" }
func (s libraryTopReference) Select(ctx *EffectContext) []uuid.UUID {
	amount := max(s.amount.Resolve(ctx), 0)
	var ids []uuid.UUID
	for _, id := range s.player.Select(ctx) {
		player := ctx.Game.GetPlayer(id)
		if player == nil {
			continue
		}
		count := amount
		if s.mill {
			count = ctx.Game.ApplyMillModifiers(id, count)
		}
		for _, card := range player.Library()[:min(max(count, 0), len(player.Library()))] {
			ids = append(ids, card.ID())
		}
	}
	return ids
}

// RememberedObject selects an object retained by a delayed effect.
func RememberedObject(index int) ObjectSelector {
	return objectReference{kind: "remembered", index: index}
}

type eventSnapshotAmount struct{}

func (eventSnapshotAmount) Text() string { return "that much" }
func (eventSnapshotAmount) Resolve(ctx *EffectContext) int {
	if ctx == nil || ctx.TriggerEvent == nil {
		return 0
	}
	return ctx.TriggerEvent.Amount
}
