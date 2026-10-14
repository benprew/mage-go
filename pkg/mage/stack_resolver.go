package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ResolveStack resolves all objects on the stack (simplified: no priority passing).
func (g *Game) ResolveStack() {
	// Move any pending triggers to the stack first (e.g. from EvtCardDrawn during draw step)
	g.PutTriggersOnStack()
	for !g.stack.IsEmpty() {
		obj := g.stack.Pop()
		g.ResolveStackObject(obj)
		// Check for new triggers after each resolution
		g.PutTriggersOnStack()
	}
}

// ResolveTopOfStack resolves just the top item on the stack.
func (g *Game) ResolveTopOfStack() {
	if g.stack.IsEmpty() {
		return
	}
	obj := g.stack.Pop()
	g.ResolveStackObject(obj)
	g.PutTriggersOnStack()
}

// zoneOfTarget reports the zone the given target id currently occupies, or
// ZoneAny if the id is a player or can't be located in any zone.
func (g *Game) zoneOfTarget(id uuid.UUID) Zone {
	if id == uuid.Nil || g.GetPlayer(id) != nil {
		return ZoneAny
	}
	if g.FindPermanent(id) != nil {
		return ZoneBattlefield
	}
	if g.stack.FindByID(id) != nil {
		return ZoneStack
	}
	if g.stack.FindBySourceID(id) != nil {
		return ZoneStack
	}
	for _, ec := range g.zones.exile {
		if ec.Card != nil && ec.Card.ID() == id {
			return ZoneExile
		}
	}
	for _, p := range g.players {
		for _, c := range p.Graveyard() {
			if c.ID() == id {
				return ZoneGraveyard
			}
		}
		for _, c := range p.Library() {
			if c.ID() == id {
				return ZoneLibrary
			}
		}
		for _, c := range p.Hand() {
			if c.ID() == id {
				return ZoneHand
			}
		}
		for _, c := range p.Ante() {
			if c.ID() == id {
				return ZoneAnte
			}
		}
	}
	return ZoneAny
}

func (g *Game) graveyardVisit(id uuid.UUID) uuid.UUID {
	for _, player := range g.players {
		for _, card := range player.Graveyard() {
			if card.ID() == id {
				if visited, ok := player.(interface{ graveyardVisit(uuid.UUID) uuid.UUID }); ok {
					return visited.graveyardVisit(id)
				}
			}
		}
	}
	return uuid.Nil
}

// Used to track game-specific state before pushing object onto stack
func (g *Game) pushStack(obj *StackObject) {
	g.chooseRandomCounterDistribution(obj)
	g.captureHandTargetCards(obj)
	g.captureExileTargetCards(obj)
	g.captureAnteTargetCards(obj)
	for _, id := range obj.Targets {
		identity, saved := obj.targetHandCards[id]
		if !saved {
			identity, saved = obj.targetExileCards[id]
		}
		if !saved {
			identity, saved = obj.targetAnteCards[id]
		}
		if saved {
			if _, captured := obj.targetIncarnations[id]; !captured {
				obj.targetIncarnations = maps.Clone(obj.targetIncarnations)
				if obj.targetIncarnations == nil {
					obj.targetIncarnations = make(map[uuid.UUID]uuid.UUID)
				}
				obj.targetIncarnations[id] = identity.visit
			}
		}
	}
	g.captureGraveyardTargetSources(obj)
	spellSourcesCopied := false
	for _, id := range obj.Targets {
		if _, saved := obj.targetExileCards[id]; saved {
			continue
		}
		if _, saved := obj.targetSpellSources[id]; saved {
			continue
		}
		if source := g.CaptureSpellDamageSource(id); source.Valid() {
			if !spellSourcesCopied {
				obj.targetSpellSources = maps.Clone(obj.targetSpellSources)
				spellSourcesCopied = true
			}
			if obj.targetSpellSources == nil {
				obj.targetSpellSources = make(map[uuid.UUID]DamageSource)
			}
			obj.targetSpellSources[id] = source
		}
	}
	// Track target zone when targeted, needed for CR 608.2b
	for _, id := range obj.Targets {
		if _, captured := obj.targetIncarnations[id]; captured {
			continue
		}
		var incarnation uuid.UUID
		if p := g.FindPermanent(id); p != nil {
			incarnation = p.incarnationID
		} else {
			incarnation = g.cardTargetVisit(id)
		}
		if incarnation != uuid.Nil {
			if obj.targetIncarnations == nil {
				obj.targetIncarnations = make(map[uuid.UUID]uuid.UUID)
			}
			obj.targetIncarnations[id] = incarnation
		}
	}
	obj.TargetZones = make(map[uuid.UUID]Zone, len(obj.Targets))
	for _, t := range obj.Targets {
		if _, saved := obj.targetHandCards[t]; saved {
			obj.TargetZones[t] = ZoneHand
		} else if _, saved := obj.targetExileCards[t]; saved {
			obj.TargetZones[t] = ZoneExile
		} else if _, saved := obj.targetAnteCards[t]; saved {
			obj.TargetZones[t] = ZoneAnte
		} else {
			obj.TargetZones[t] = g.zoneOfTarget(t)
		}
	}

	g.stack.Push(obj)
}

// Check if target is still legal at resolution time. Used in stack responses and
// resolutions.
func (g *Game) isTargetStillLegal(targetID uuid.UUID, sourceCard Card, controller uuid.UUID, expectedZone Zone, spec Target) bool {
	return g.isTargetStillLegalWithX(targetID, sourceCard, controller, expectedZone, spec, g.resolution.X())
}

func (g *Game) isTargetStillLegalWithX(targetID uuid.UUID, sourceCard Card, controller uuid.UUID, expectedZone Zone, spec Target, x int) bool {
	if g.GetPlayer(targetID) != nil {
		return spec == nil || slices.Contains(g.possibleTargetsWithX(spec, controller, sourceCard, x), targetID)
	}
	// CR 608.2b: "A target that's no longer in the zone it was in when it was
	// targeted is illegal."
	if g.zoneOfTarget(targetID) != expectedZone {
		return false
	}
	if perm := g.FindPermanent(targetID); perm != nil {
		if !perm.CanBeTargetedBy(sourceCard, controller, g) {
			return false
		}
	}
	if spec != nil && !slices.Contains(g.possibleTargetsWithX(spec, controller, sourceCard, x), targetID) {
		return false
	}
	return true
}

// ResolveStackObject resolves a single stack object.
func (g *Game) ResolveStackObject(obj *StackObject) {
	g.resolveStackObject(obj)
}

func (g *Game) resolveStackObject(obj *StackObject) (produced int, colors []Color) {
	referenceDepth := len(g.stack.resolvingReferenceObjects)
	g.stack.resolvingReferenceObjects = append(g.stack.resolvingReferenceObjects, obj)
	previousMana, previousTracking := g.resolution.manaProduced, g.resolution.manaTracking
	previousColors := g.resolution.manaColors
	g.resolution.manaColors = nil
	g.resolution.manaProduced, g.resolution.manaTracking = 0, true
	defer func() {
		produced = g.resolution.manaProduced
		colors = g.resolution.manaColors
		g.resolution.manaColors = previousColors
		g.resolution.manaProduced, g.resolution.manaTracking = previousMana, previousTracking
	}()
	oldIncarnation, oldCaptured := g.resolution.sourceIncarnation, g.resolution.sourceCaptured
	g.resolution.sourceIncarnation, g.resolution.sourceCaptured = obj.sourceIncarnation, true
	defer func() { g.resolution.sourceIncarnation, g.resolution.sourceCaptured = oldIncarnation, oldCaptured }()
	previousState := g.triggers.resolvingState
	g.triggers.resolvingState = obj.stateKey
	defer g.CheckStateBasedActions()
	defer func() {
		g.triggers.resolvingState = previousState
		g.CheckStateTriggers()
	}()
	defer func() { g.stack.resolvingReferenceObjects = g.stack.resolvingReferenceObjects[:referenceDepth] }()

	// Check for fizzle: if the spell/ability has targets but all are now illegal,
	// it fails to resolve (MTG rule 608.2b)

	defer g.ClearSacrificed()
	defer g.resolution.SetColorOverride(obj.SourceID, obj.ColorOverride)()

	// CR 608.2b: a spell/ability with target(s) fails to resolve only if ALL of
	// them are illegal at resolution. uuid.Nil entries are positional
	// placeholders for target slots that had no legal candidate — not real
	// targets — so they never count as "has a target", and every slot is kept
	// in place (illegal ones nulled) so effects that read targets by index
	// (e.g. Hungry Flames) stay aligned.
	resolvedTargets := make([]uuid.UUID, len(obj.Targets))
	targetQuery := g.targetEventQuery(obj.capturedPlayers, obj.capturedPermanents).targetBindingsQuery(obj.TargetSpecs, obj.Targets, obj.targetIncarnations, obj.XValue)
	sourceCard := g.resolutionTargetSource(obj)
	hasRealTarget := false
	anyLegal := false
	for i, t := range obj.Targets {
		if t != uuid.Nil {
			hasRealTarget = true
			var spec Target
			if i < len(obj.TargetSpecs) {
				spec = obj.TargetSpecs[i]
			}
			incarnationLegal := true
			if incarnation, ok := obj.targetIncarnations[t]; ok {
				if obj.TargetZones[t] == ZoneGraveyard || obj.TargetZones[t] == ZoneHand || obj.TargetZones[t] == ZoneExile || obj.TargetZones[t] == ZoneAnte {
					incarnationLegal = g.cardTargetVisit(t) == incarnation
				} else {
					p := g.FindPermanent(t)
					incarnationLegal = p != nil && p.incarnationID == incarnation
				}
			}
			if incarnationLegal && targetQuery.isTargetStillLegalWithX(t, sourceCard, obj.Controller, obj.TargetZones[t], spec, obj.XValue) {
				resolvedTargets[i] = t
				anyLegal = true
				continue
			}
		}
		resolvedTargets[i] = uuid.Nil
	}

	if hasRealTarget && !anyLegal {
		g.cleanupSpell(obj)
		return
	}

	defer g.resolution.Begin(obj)()
	for _, eff := range obj.Effects {
		_ = applyEffectWithCardSetCaptures(g, eff, obj.SourceID, obj.Controller, resolvedTargets, obj.capturedNumbers, obj.capturedPermanents, obj.capturedCards, obj.capturedCardSets)
	}
	g.resolution.ClearDistributions()

	// Permanent spell copies must use the normal entry path (CR 707.10f).
	permanentSpell := obj.Card != nil && !obj.IsAbility && (obj.Card.HasType(TypeCreature) || obj.Card.HasType(TypeArtifact) || obj.Card.HasType(TypeEnchantment) || obj.Card.HasType(TypePlaneswalker) || obj.Card.HasType(TypeLand))
	if obj.leftStack || (obj.IsCopy && !permanentSpell) {
		return
	}

	// If this was a spell (not an ability), put the card in the graveyard
	if obj.Card != nil && !obj.IsAbility {
		// Permanents go to the battlefield instead
		if permanentSpell {
			card := obj.Card
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.landAuraSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildLandAuraSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.staticActivationText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildStaticActivationSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.protectionPTSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildProtectionPTSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.intrinsicCastText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildIntrinsicCastSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.intrinsicSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildIntrinsicCreatureSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.copyEntrySpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildCopyEntrySpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.globalStaticSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildGlobalStaticSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.spellCastSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildSpellCastSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.upkeepSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildUpkeepSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.manaTapSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildManaTapSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			if base, ok := obj.spellTextDefinition.(*BaseCard); ok && base.staticEventSpellText != 0 && base.textDefinition != nil && len(base.textDefinition.changes) != 0 {
				continued, err := buildMixedStaticEventSpellEntry(obj)
				if err != nil {
					g.cleanupSpell(obj)
					return
				}
				card = continued
			}
			colorsSpent := obj.CastContext.DistinctColorsSpent()
			if obj.IsCopy {
				card = newPermanentSpellCopyCard(card)
				colorsSpent = 0
			}
			var hostID uuid.UUID
			if obj.Card.HasType(TypeEnchantment) && slices.Contains(obj.Card.SubTypes(), "Aura") && len(resolvedTargets) > 0 {
				hostID = resolvedTargets[0]
			}
			g.putPermanentSpellOnBattlefield(obj, card, colorsSpent, hostID)
			return
		}

		g.cleanupSpell(obj)
	}
	return
}

// Move spell to graveyard/exile as appropriate. Used for resolving cast
// instants/sorceries and fizzled spells
func (g *Game) cleanupSpell(obj *StackObject) {
	// do nothing for non-spells
	if obj.Card == nil || obj.IsAbility || obj.leftStack {
		return
	}

	// Copies of spells cease to exist (CR 707.10) — no graveyard, no exile.
	if obj.IsCopy {
		return
	}

	owner := obj.Card.Owner()
	if owner == uuid.Nil {
		owner = obj.Controller
	}
	if obj.ExileOnLeaveStack || g.spellMarkedExileInstead(obj) {
		g.ExileCard(obj.Card, obj.Card.ID())
	} else {
		p := g.GetPlayer(owner)
		if p != nil {
			p.AddToGraveyard(obj.Card)
		}
	}
}
