package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// For cloning UUID maps with generic
type UUIDMap[V any] map[uuid.UUID]V

// Clone creates a deep copy of the game state for AI search.
// Card objects and Effect interfaces are shared (immutable during play).
// Battlefield permanents are shared copy-on-write; other mutable state is
// deep-copied.
// Interactive callbacks (OnPriority, etc.) are nil'd in the clone.
// Players are wrapped in SearchPlayer for non-interactive choice defaults.
func (g *Game) Clone() *Game {
	c := &Game{
		anteEnabled: g.anteEnabled,
		anteSettled: g.anteSettled,
		stopped:     g.stopped,
	}
	// Saved payment frames do not change; each payment copies its frame.
	c.playerPayments = slices.Clone(g.playerPayments)
	c.turns = g.turns.Clone()
	c.zones = g.zones.Clone()
	c.resolution = g.resolution.Clone()
	c.trackers = g.trackers.Clone()
	c.random = g.random.Clone()
	c.triggers = g.triggers.Clone()
	c.mana = g.mana.Clone()
	c.damage = g.damage.Clone()
	c.publicReveals = clonePublicReveals(g.publicReveals)
	if g.handLooks != nil {
		c.handLooks = make(map[uuid.UUID][]PublicCardReveal, len(g.handLooks))
		for viewer, cards := range g.handLooks {
			c.handLooks[viewer] = clonePublicReveals(cards)
		}
	}
	if g.libraryLooks != nil {
		c.libraryLooks = make(map[uuid.UUID][]PublicCardReveal, len(g.libraryLooks))
		for viewer, cards := range g.libraryLooks {
			c.libraryLooks[viewer] = clonePublicReveals(cards)
		}
	}
	c.lifeGainModifiers = slices.Clone(g.lifeGainModifiers)
	c.millModifiers = slices.Clone(g.millModifiers)
	c.originalOwners = cloneUUIDMap(g.originalOwners)
	if c.originalOwners == nil {
		c.originalOwners = make(map[uuid.UUID]uuid.UUID)
	}
	c.anteResult = append([]OwnershipChange(nil), g.anteResult...)

	// Deep copy players, wrapping in SearchPlayer for non-interactive choices.
	c.players = make([]Player, len(g.players))
	for i, p := range g.players {
		c.players[i] = clonePlayer(p)
	}

	// Deep copy stack.
	c.stack = cloneStack(g.stack)
	c.bindHandStack()

	// Deep copy combat.
	c.combat = cloneCombat(g.combat)

	// Deep copy effect manager.
	c.effects = cloneEffectManager(g.effects)

	c.customState = cloneCustomState(g.customState)

	// Interactive callbacks are nil'd — search clones don't call back to UI.
	// OnPriority, AfterPriorityAction, BeforeStackResolve all remain nil.

	return c
}

// clonePlayer creates a SearchPlayer wrapping a deep copy of the player's BasePlayer state.
func clonePlayer(p Player) *SearchPlayer {
	bp, ok := p.(*BasePlayer)
	if !ok {
		// Try to extract BasePlayer from SearchPlayer or other wrappers.
		if sp, ok := p.(*SearchPlayer); ok {
			bp = sp.BasePlayer
		} else {
			// Fallback: create a minimal BasePlayer with matching state.
			bp = extractBasePlayer(p)
		}
	}
	return NewSearchPlayer(cloneBasePlayer(bp))
}

// extractBasePlayer creates a BasePlayer from any Player interface.
func extractBasePlayer(p Player) *BasePlayer {
	bp := &BasePlayer{
		id:              p.PlayerID(),
		name:            p.Name(),
		life:            p.Life(),
		drewFromEmpty:   p.DrewFromEmpty(),
		manaPool:        NewManaPool(),
		poisonCounters:  p.PoisonCounters(),
		lastDrawnCardID: p.LastDrawnCardID(),
	}
	if !p.IsAlive() && p.Life() > 0 {
		bp.lost = true
	}
	// Copy zones.
	bp.hand = cloneZoneCards(p.Hand())
	if state, ok := p.(interface{ handIsPublic() bool }); ok {
		bp.handPublic = state.handIsPublic()
	}
	if history, ok := p.(interface {
		handDepartures() map[uuid.UUID]handDeparture
	}); ok {
		bp.departedHandSources = history.handDepartures()
	}
	if visited, ok := p.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
		bp.handVisits = make(map[uuid.UUID]uuid.UUID, len(bp.hand))
		for _, card := range bp.hand {
			bp.handVisits[card.ID()] = visited.handVisit(card.ID())
		}
	}
	bp.graveyard = cloneZoneCards(p.Graveyard())
	if history, ok := p.(interface {
		graveyardDepartures() map[uuid.UUID]DamageSource
	}); ok {
		bp.departedGraveyardSources = history.graveyardDepartures()
	}
	if visited, ok := p.(interface{ graveyardVisit(uuid.UUID) uuid.UUID }); ok {
		bp.graveyardVisits = make(map[uuid.UUID]uuid.UUID, len(bp.graveyard))
		for _, card := range bp.graveyard {
			bp.graveyardVisits[card.ID()] = visited.graveyardVisit(card.ID())
		}
	}
	bp.library = cloneCardSlice(p.Library())
	if order, ok := p.(interface{ libraryOrderSnapshot() []uuid.UUID }); ok {
		bp.libraryOrder = order.libraryOrderSnapshot()
	}
	if state, ok := p.(interface{ libraryTopIsPublic() bool }); ok {
		bp.libraryTopPublic = state.libraryTopIsPublic()
	}
	if visited, ok := p.(interface{ libraryVisit(uuid.UUID) uuid.UUID }); ok {
		bp.libraryVisits = make(map[uuid.UUID]uuid.UUID, len(bp.library))
		for _, card := range bp.library {
			bp.libraryVisits[card.ID()] = visited.libraryVisit(card.ID())
		}
	}
	bp.ante = cloneCardSlice(p.Ante())
	if visitor, ok := p.(interface{ anteVisit(uuid.UUID) uuid.UUID }); ok {
		bp.anteVisits = make(map[uuid.UUID]uuid.UUID, len(bp.ante))
		for _, card := range bp.ante {
			bp.anteVisits[card.ID()] = visitor.anteVisit(card.ID())
		}
	}
	// Copy mana pool.
	bp.manaPool.RestorePool(p.ManaPool().SnapshotPool())
	if len(p.ManaPool().ManaConversions) > 0 {
		bp.manaPool.ManaConversions = make(map[Color]Color)
		maps.Copy(bp.manaPool.ManaConversions, p.ManaPool().ManaConversions)
	}
	return bp
}

// cloneBasePlayer deep copies a BasePlayer, preserving the same UUID.
func cloneBasePlayer(bp *BasePlayer) *BasePlayer {
	clone := &BasePlayer{
		id:              bp.id,
		name:            bp.name,
		life:            bp.life,
		lost:            bp.lost,
		drewFromEmpty:   bp.drewFromEmpty,
		lastDrawnCardID: bp.lastDrawnCardID,
		poisonCounters:  bp.poisonCounters,
		manaPool:        NewManaPool(),
	}
	// Mutable zone facts must stay separate in each game.
	clone.hand = cloneZoneCards(bp.hand)
	clone.departedHandSources = bp.departedHandSources
	clone.handPublic = bp.handPublic
	clone.handVisits = bp.handVisits
	clone.graveyard = cloneZoneCards(bp.graveyard)
	clone.departedGraveyardSources = bp.departedGraveyardSources
	clone.graveyardVisits = bp.graveyardVisits
	clone.library = cloneCardSlice(bp.library)
	clone.libraryVisits = bp.libraryVisits
	clone.libraryOrder = bp.libraryOrder
	clone.libraryTopPublic = bp.libraryTopPublic
	clone.ante = cloneCardSlice(bp.ante)
	clone.anteVisits = bp.anteVisits
	// Copy mana pool state.
	clone.manaPool.RestorePool(bp.manaPool.SnapshotPool())
	if len(bp.manaPool.ManaConversions) > 0 {
		clone.manaPool.ManaConversions = make(map[Color]Color)
		maps.Copy(clone.manaPool.ManaConversions, bp.manaPool.ManaConversions)
	}
	return clone
}

// clonePermanentInto deep-copies src into dst (which may be a freshly-allocated
// struct or a slot inside a slab). Card refs, Ability interface refs, and
// BasePTOverride/ColorOverride pointer contents are deep-copied; slices are
// copied only when non-empty.
func clonePermanentInto(dst, src *Permanent) {
	*dst = Permanent{
		Card:                       src.Card, // shared
		incarnationID:              src.incarnationID,
		attachedIncarnation:        src.attachedIncarnation,
		computedController:         src.computedController,
		baseController:             src.baseController,
		Tapped:                     src.Tapped,
		PhasedOut:                  src.PhasedOut,
		Damage:                     src.Damage,
		AttachedTo:                 src.AttachedTo,
		IsToken:                    src.IsToken,
		FaceDown:                   src.FaceDown,
		ChosenColor:                src.ChosenColor,
		ChosenPlayer:               src.ChosenPlayer,
		ChosenSubtype:              src.ChosenSubtype,
		ChosenBasicLandType:        src.ChosenBasicLandType,
		ControlledPermanent:        src.ControlledPermanent,
		turnControlGained:          src.turnControlGained,
		controlGainedTurns:         maps.Clone(src.controlGainedTurns),
		StoredValue:                src.StoredValue,
		CreatedBy:                  src.CreatedBy,
		printedAbilitiesSuppressed: src.printedAbilitiesSuppressed,
		revisedPrintedTriggers:     src.revisedPrintedTriggers,
		powerBonus:                 src.powerBonus,
		toughBonus:                 src.toughBonus,
		ptSwitched:                 src.ptSwitched,
		DeathtouchDamage:           src.DeathtouchDamage,
		Counters:                   src.Counters, // fixed-size array: value copy
		baseAttrs:                  src.baseAttrs,
		printedAbilityAttrs:        src.printedAbilityAttrs,
		grantedAttrs:               src.grantedAttrs,
	}
	for family := range subtypeFamilyCount {
		dst.subtypeFamilyOverrideSet[family] = src.subtypeFamilyOverrideSet[family]
		dst.subtypeFamilyOverrides[family] = append([]string(nil), src.subtypeFamilyOverrides[family]...)
		dst.subtypeFamilyAdditions[family] = append([]string(nil), src.subtypeFamilyAdditions[family]...)
	}
	// Deep copy attachments.
	if len(src.Attachments) > 0 {
		dst.Attachments = make([]uuid.UUID, len(src.Attachments))
		copy(dst.Attachments, src.Attachments)
	}
	// RuntimeAbilities: copy-on-write. Share the backing array with src but
	// force cap == len via the three-index slice form, so any subsequent
	// append on either side allocates a new array. All current mutation
	// sites are either `= nil`, `= append(...)`, or `= freshSlice` — never
	// an index assignment — so this sharing is safe.
	if n := len(src.RuntimeAbilities); n > 0 {
		dst.RuntimeAbilities = src.RuntimeAbilities[:n:n]
	}
	// Deep copy subtype override.
	if len(src.SubTypeOverride) > 0 {
		dst.SubTypeOverride = make([]string, len(src.SubTypeOverride))
		copy(dst.SubTypeOverride, src.SubTypeOverride)
	}
	if len(src.SubTypeAdditions) > 0 {
		dst.SubTypeAdditions = make([]string, len(src.SubTypeAdditions))
		copy(dst.SubTypeAdditions, src.SubTypeAdditions)
	}
	// Deep copy cdaPower and cdaToughness if non-nil.
	if src.cdaPower != nil {
		v := *src.cdaPower
		dst.cdaPower = &v
	}
	if src.cdaToughness != nil {
		v := *src.cdaToughness
		dst.cdaToughness = &v
	}
	// Deep copy BasePTOverride if non-nil (it's a *[2]int).
	if src.BasePTOverride != nil {
		v := *src.BasePTOverride
		dst.BasePTOverride = &v
	}
	// Deep copy ColorOverride if non-nil.
	if src.ColorOverride != nil {
		colors := make([]Color, len(*src.ColorOverride))
		copy(colors, *src.ColorOverride)
		dst.ColorOverride = &colors
	}
}

// cloneStack deep copies the Stack.
func cloneStack(s *Stack) *Stack {
	if s == nil {
		return NewStack()
	}
	clone := &Stack{spellSources: maps.Clone(s.spellSources), spellManaValues: maps.Clone(s.spellManaValues), spellControllers: maps.Clone(s.spellControllers)}
	if len(s.resolvingReferenceObjects) > 0 {
		clone.resolvingReferenceObjects = make([]*StackObject, len(s.resolvingReferenceObjects))
		for i, object := range s.resolvingReferenceObjects {
			clone.resolvingReferenceObjects[i] = cloneStackObject(object)
		}
	}
	if len(s.objects) > 0 {
		clone.objects = make([]*StackObject, len(s.objects))
		for i, obj := range s.objects {
			clone.objects[i] = cloneStackObject(obj)
		}
	}
	return clone
}

// cloneStackObject deep copies a StackObject.
func cloneStackObject(obj *StackObject) *StackObject {
	clone := &StackObject{
		targetIncarnations:     obj.targetIncarnations,
		revealedHandCards:      slices.Clone(obj.revealedHandCards),
		ownsHandReveals:        obj.ownsHandReveals,
		targetSpellSources:     obj.targetSpellSources,
		targetGraveyardSources: obj.targetGraveyardSources,
		targetHandCards:        obj.targetHandCards,
		targetExileCards:       obj.targetExileCards,
		targetAnteCards:        obj.targetAnteCards,
		boundSpellSources:      obj.boundSpellSources,
		sourceIncarnation:      obj.sourceIncarnation,
		capturedNumbers:        maps.Clone(obj.capturedNumbers),
		capturedPlayers:        maps.Clone(obj.capturedPlayers),
		capturedCards:          maps.Clone(obj.capturedCards),
		capturedCardSets:       cloneCardsBindings(obj.capturedCardSets),
		capturedPermanents:     maps.Clone(obj.capturedPermanents),
		stateKey:               obj.stateKey,
		ID:                     obj.ID,
		Card:                   obj.Card, // shared Card ref
		Controller:             obj.Controller,
		SourceID:               obj.SourceID,
		TargetSource:           obj.TargetSource,
		IsAbility:              obj.IsAbility,
		activatedAbility:       obj.activatedAbility,
		abilitySourceVisit:     obj.abilitySourceVisit,
		XValue:                 obj.XValue,
		ModeChoice:             obj.ModeChoice,
		EventAmount:            obj.EventAmount,
		EventSourceID:          obj.EventSourceID,
		IsCopy:                 obj.IsCopy,
		CastZone:               obj.CastZone,
		CastContext:            obj.CastContext.clone(),
	}
	if obj.ColorOverride != nil {
		colors := append([]Color(nil), (*obj.ColorOverride)...)
		clone.ColorOverride = &colors
	}
	if len(obj.ModalTargets) > 0 || len(obj.Targets) > 0 {
		if len(obj.ModalTargets) > 0 {
			clone.ModalTargets = make([][]uuid.UUID, len(obj.ModalTargets))
		}
		count := len(obj.Targets)
		for _, targets := range obj.ModalTargets {
			count += len(targets)
		}
		// Store target groups together to limit clone allocations.
		storage := make([]uuid.UUID, count)
		if len(obj.Targets) > 0 {
			copy(storage, obj.Targets)
			clone.Targets = storage[:len(obj.Targets):len(obj.Targets)]
			storage = storage[len(obj.Targets):]
		}
		for i, targets := range obj.ModalTargets {
			if len(targets) > 0 {
				copy(storage, targets)
				clone.ModalTargets[i] = storage[:len(targets):len(targets)]
				storage = storage[len(targets):]
			}
		}
	}
	// Share Effect interface refs.
	if len(obj.Effects) > 0 {
		clone.Effects = make([]Effect, len(obj.Effects))
		copy(clone.Effects, obj.Effects)
	}
	if len(obj.TargetSpecs) > 0 {
		clone.TargetSpecs = append([]Target(nil), obj.TargetSpecs...)
	}
	if len(obj.DamageDistribution) > 0 {
		clone.DamageDistribution = make(map[uuid.UUID]int, len(obj.DamageDistribution))
		maps.Copy(clone.DamageDistribution, obj.DamageDistribution)
	}
	if len(obj.CounterDistribution) > 0 {
		clone.CounterDistribution = make(map[uuid.UUID]int, len(obj.CounterDistribution))
		maps.Copy(clone.CounterDistribution, obj.CounterDistribution)
	}
	if len(obj.TargetZones) > 0 {
		clone.TargetZones = make(map[uuid.UUID]Zone, len(obj.TargetZones))
		maps.Copy(clone.TargetZones, obj.TargetZones)
	}
	return clone
}

// cloneCombat deep copies the Combat struct.
func cloneCombat(c *Combat) *Combat {
	if c == nil {
		return nil
	}
	clone := &Combat{
		Attackers:            cloneUUIDBoolMap(c.Attackers),
		AttackerIncarnations: maps.Clone(c.AttackerIncarnations),
		FirstStruck:          cloneUUIDBoolMap(c.FirstStruck),
		AttackedAlone:        c.AttackedAlone,
		BlockedAlone:         c.BlockedAlone,
	}
	// Deep copy groups.
	if len(c.Groups) > 0 {
		clone.Groups = make([]*CombatGroup, len(c.Groups))
		for i, g := range c.Groups {
			cg := &CombatGroup{
				AttackerID: g.AttackerID,
				DefenderID: g.DefenderID,
				Blocked:    g.Blocked,
				Unblocked:  g.Unblocked,
				blockedBy:  maps.Clone(g.blockedBy),
			}
			if len(g.BlockerIDs) > 0 {
				cg.BlockerIDs = make([]uuid.UUID, len(g.BlockerIDs))
				copy(cg.BlockerIDs, g.BlockerIDs)
				cg.BlockerIncarnations = slices.Clone(g.BlockerIncarnations)
			}
			clone.Groups[i] = cg
		}
	}
	// Deep copy bands (UUID -> []UUID).
	if len(c.Bands) > 0 {
		clone.Bands = make(map[uuid.UUID][]uuid.UUID, len(c.Bands))
		for k, v := range c.Bands {
			members := make([]uuid.UUID, len(v))
			copy(members, v)
			clone.Bands[k] = members
		}
	} else {
		clone.Bands = make(map[uuid.UUID][]uuid.UUID)
	}
	return clone
}

// cloneEffectManager deep copies the EffectManager.
func cloneEffectManager(em *EffectManager) *EffectManager {
	clone := &EffectManager{
		// attrDeltas and blockPairRestrictions are recomputed each Apply() cycle.
		attrDeltas: make(map[uuid.UUID]map[Attr]int),
	}
	// A clone must keep these rules for queries before the next layer pass.
	clone.noCombatAssignmentRules = maps.Clone(em.noCombatAssignmentRules)
	if len(em.blockAnyNumberRules) > 0 {
		clone.blockAnyNumberRules = make(map[uuid.UUID]bool, len(em.blockAnyNumberRules))
		maps.Copy(clone.blockAnyNumberRules, em.blockAnyNumberRules)
	}
	if len(em.blockEachAttackerRules) > 0 {
		clone.blockEachAttackerRules = make(map[uuid.UUID]int, len(em.blockEachAttackerRules))
		maps.Copy(clone.blockEachAttackerRules, em.blockEachAttackerRules)
	}
	if len(em.allCreaturesMustBlockRules) > 0 {
		clone.allCreaturesMustBlockRules = make(map[uuid.UUID][]PermanentFilter, len(em.allCreaturesMustBlockRules))
		for id, filters := range em.allCreaturesMustBlockRules {
			clone.allCreaturesMustBlockRules[id] = slices.Clone(filters)
		}
	}
	if len(em.cantBeBlockedExceptByRules) > 0 {
		clone.cantBeBlockedExceptByRules = make(map[uuid.UUID][]PermanentFilter, len(em.cantBeBlockedExceptByRules))
		for id, filters := range em.cantBeBlockedExceptByRules {
			clone.cantBeBlockedExceptByRules[id] = slices.Clone(filters)
		}
	}
	if len(em.canBlockOnlyRules) > 0 {
		clone.canBlockOnlyRules = make(map[uuid.UUID][]PermanentFilter, len(em.canBlockOnlyRules))
		for id, filters := range em.canBlockOnlyRules {
			clone.canBlockOnlyRules[id] = slices.Clone(filters)
		}
	}
	// Copy continuous effects slice. Effects are shared interface values (they
	// receive *Game as a parameter and are otherwise stateless), except those
	// implementing deepCloneableEffect, which carry mutable per-game state and
	// must be deep-copied so a search clone can't corrupt the original.
	if len(em.effects) > 0 {
		clone.effects = make([]ContinuousEffect, len(em.effects))
		for i, e := range em.effects {
			if dc, ok := e.(deepCloneableEffect); ok {
				clone.effects[i] = dc.cloneEffect()
			} else {
				clone.effects[i] = e
			}
		}
	}
	// Deep copy replacement effects (they hold mutable state).
	clone.replacements = cloneReplacementSlice(em.replacements)
	clone.cycleReplacements = cloneReplacementSlice(em.cycleReplacements)
	// Clone subsystems.
	clone.Rules = cloneGameRules(em.Rules)
	if len(em.bandsWithOther) > 0 {
		clone.bandsWithOther = make(map[capturedPermanent][]PermanentFilter, len(em.bandsWithOther))
		for subject, qualities := range em.bandsWithOther {
			clone.bandsWithOther[subject] = slices.Clone(qualities)
		}
	}

	return clone
}

// cloneReplacementSlice deep copies a slice of ReplacementEffect using Clone().
func cloneReplacementSlice(src []ReplacementEffect) []ReplacementEffect {
	if len(src) == 0 {
		return nil
	}
	dst := make([]ReplacementEffect, len(src))
	for i, r := range src {
		dst[i] = r.Clone()
	}
	return dst
}

// cloneGameRules deep copies the GameRules.
func cloneGameRules(gr *GameRules) *GameRules {
	clone := &GameRules{
		LandUntapMax:       gr.LandUntapMax,
		ArtifactUntapMax:   gr.ArtifactUntapMax,
		CreatureUntapMax:   gr.CreatureUntapMax,
		UnlimitedLandPlays: gr.UnlimitedLandPlays,
	}
	// Clone all maps.
	clone.ManaConversion = cloneColorColorMap(gr.ManaConversion)
	clone.SpellCostIncreases = cloneColorIntMap(gr.SpellCostIncreases)
	clone.spellCostTaxes = slices.Clone(gr.spellCostTaxes)
	clone.activationCostModifiers = slices.Clone(gr.activationCostModifiers)
	clone.SpellCostReductions = cloneColorIntMap(gr.SpellCostReductions)
	clone.SpellTypeCostReductions = cloneCardTypeIntMap(gr.SpellTypeCostReductions)
	clone.sanctuaryActive = cloneUUIDBoolMap(gr.sanctuaryActive)
	clone.lichActive = cloneUUIDUUIDMap(gr.lichActive)
	clone.skipNextDraw = cloneUUIDBoolMap(gr.skipNextDraw)
	clone.channelActive = cloneUUIDBoolMap(gr.channelActive)
	clone.minimumLife = cloneUUIDBoolMap(gr.minimumLife)
	clone.maxHandSize = cloneUUIDIntMap(gr.maxHandSize)
	clone.revealedTopCard = maps.Clone(gr.revealedTopCard)
	clone.revealedHand = maps.Clone(gr.revealedHand)
	clone.NullifiedLandwalks = cloneAttrBoolMap(gr.NullifiedLandwalks)
	clone.ActivationCostReductions = cloneUUIDIntMap(gr.ActivationCostReductions)
	if len(gr.SpellCostReducers) > 0 {
		clone.SpellCostReducers = make([]SpellCostReducer, len(gr.SpellCostReducers))
		copy(clone.SpellCostReducers, gr.SpellCostReducers)
	}
	// Copy expansion cast blocks.
	if len(gr.expansionCastBlock) > 0 {
		clone.expansionCastBlock = make([]string, len(gr.expansionCastBlock))
		copy(clone.expansionCastBlock, gr.expansionCastBlock)
	}
	return clone
}

// --- Map clone helpers ---

// cloneCardSlice shares the backing array with the source but forces cap == len
// so any subsequent append on either side allocates a new backing array.
// Mutation paths that would otherwise touch shared memory (middle-element
// removal in RemoveFromHand/Ante, ShuffleLibrary) are required to
// allocate fresh slices — see player.go.
func cloneCardSlice(src []Card) []Card {
	if len(src) == 0 {
		return nil
	}
	return src[:len(src):len(src)]
}

func cloneUUIDBoolMap(src map[uuid.UUID]bool) map[uuid.UUID]bool {
	if src == nil {
		return make(map[uuid.UUID]bool)
	}
	dst := make(map[uuid.UUID]bool, len(src))
	maps.Copy(dst, src)
	return dst
}

// cloneUUIDMap returns nil for nil or empty input, avoiding the
// per-Clone allocation of an empty map header. Use this for fields whose
// write paths lazy-initialize the map on first write.
func cloneUUIDMap[V any](src UUIDMap[V]) UUIDMap[V] {
	if len(src) == 0 {
		return nil
	}
	dst := make(UUIDMap[V], len(src))
	maps.Copy(dst, src)
	return dst
}

// cloneUUIDColorMap deep-copies a nested per-player color tally so a search
// clone can't mutate the real per-duel counters through a shared inner map.
func cloneUUIDColorMap(src map[uuid.UUID]map[Color]int) map[uuid.UUID]map[Color]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[uuid.UUID]map[Color]int, len(src))
	for k, inner := range src {
		ic := make(map[Color]int, len(inner))
		maps.Copy(ic, inner)
		dst[k] = ic
	}
	return dst
}

// cloneUUIDCardTypeMap deep-copies a nested per-player card-type tally.
func cloneUUIDCardTypeMap(src map[uuid.UUID]map[CardType]int) map[uuid.UUID]map[CardType]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[uuid.UUID]map[CardType]int, len(src))
	for k, inner := range src {
		ic := make(map[CardType]int, len(inner))
		maps.Copy(ic, inner)
		dst[k] = ic
	}
	return dst
}

func cloneUUIDIntMap(src map[uuid.UUID]int) map[uuid.UUID]int {
	if src == nil {
		return make(map[uuid.UUID]int)
	}
	dst := make(map[uuid.UUID]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneUUIDUUIDMap(src map[uuid.UUID]uuid.UUID) map[uuid.UUID]uuid.UUID {
	if src == nil {
		return make(map[uuid.UUID]uuid.UUID)
	}
	dst := make(map[uuid.UUID]uuid.UUID, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneNestedUUIDMap(src map[uuid.UUID]map[uuid.UUID]bool) map[uuid.UUID]map[uuid.UUID]bool {
	if src == nil {
		return make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	dst := make(map[uuid.UUID]map[uuid.UUID]bool, len(src))
	for k, v := range src {
		inner := make(map[uuid.UUID]bool, len(v))
		maps.Copy(inner, v)
		dst[k] = inner
	}
	return dst
}

func cloneBlockedThisTurn(src map[uuid.UUID][]uuid.UUID) map[uuid.UUID][]uuid.UUID {
	if src == nil {
		return make(map[uuid.UUID][]uuid.UUID)
	}
	dst := make(map[uuid.UUID][]uuid.UUID, len(src))
	for k, v := range src {
		s := make([]uuid.UUID, len(v))
		copy(s, v)
		dst[k] = s
	}
	return dst
}

func cloneCustomState(src map[string]any) map[string]any {
	if src == nil {
		return make(map[string]any)
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if st, ok := v.(cloneableCustomState); ok {
			dst[k] = st.cloneCustomState()
			continue
		}
		dst[k] = v
	}
	return dst
}

type cloneableCustomState interface {
	cloneCustomState() any
}

func cloneParadigmState(src *paradigmState) *paradigmState {
	if src == nil {
		return nil
	}
	return &paradigmState{
		Resolved:            cloneNestedUUIDStringBoolMap(src.Resolved),
		ExiledIDs:           cloneNestedUUIDStringUUIDMap(src.ExiledIDs),
		RecurringRegistered: cloneNestedUUIDStringBoolMap(src.RecurringRegistered),
	}
}

func cloneNestedUUIDStringBoolMap(src map[uuid.UUID]map[string]bool) map[uuid.UUID]map[string]bool {
	if src == nil {
		return make(map[uuid.UUID]map[string]bool)
	}
	dst := make(map[uuid.UUID]map[string]bool, len(src))
	for k, inner := range src {
		innerDst := make(map[string]bool, len(inner))
		maps.Copy(innerDst, inner)
		dst[k] = innerDst
	}
	return dst
}

func cloneNestedUUIDStringUUIDMap(src map[uuid.UUID]map[string]uuid.UUID) map[uuid.UUID]map[string]uuid.UUID {
	if src == nil {
		return make(map[uuid.UUID]map[string]uuid.UUID)
	}
	dst := make(map[uuid.UUID]map[string]uuid.UUID, len(src))
	for k, inner := range src {
		innerDst := make(map[string]uuid.UUID, len(inner))
		maps.Copy(innerDst, inner)
		dst[k] = innerDst
	}
	return dst
}

func cloneColorColorMap(src map[Color]Color) map[Color]Color {
	if src == nil {
		return make(map[Color]Color)
	}
	dst := make(map[Color]Color, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneColorIntMap(src map[Color]int) map[Color]int {
	if src == nil {
		return make(map[Color]int)
	}
	dst := make(map[Color]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneCardTypeIntMap(src map[CardType]int) map[CardType]int {
	if src == nil {
		return make(map[CardType]int)
	}
	dst := make(map[CardType]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func cloneAttrBoolMap(src map[Attr]bool) map[Attr]bool {
	if src == nil {
		return make(map[Attr]bool)
	}
	dst := make(map[Attr]bool, len(src))
	maps.Copy(dst, src)
	return dst
}
