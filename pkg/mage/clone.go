package mage

import (
	"maps"

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
		turn:                      g.turn,
		step:                      g.step,
		activePlayer:              g.activePlayer,
		currentX:                  g.currentX,
		currentMode:               g.currentMode,
		currentEventAmount:        g.currentEventAmount,
		currentEventSourceID:      g.currentEventSourceID,
		resolvingCard:             g.resolvingCard, // Card ref shared
		landsPlayedThisTurn:       g.landsPlayedThisTurn,
		creatureDeathsThisTurn:    g.creatureDeathsThisTurn,
		cleanupPriorityRounds:     g.cleanupPriorityRounds,
		stopped:                   g.stopped,
		resolvingCombatDamage:     g.resolvingCombatDamage,
		cardsPutIntoExileThisTurn: g.cardsPutIntoExileThisTurn,
	}

	// Deep copy players, wrapping in SearchPlayer for non-interactive choices.
	c.players = make([]Player, len(g.players))
	for i, p := range g.players {
		c.players[i] = clonePlayer(p)
	}

	// Share battlefield permanents copy-on-write. Cap-limiting forces appends in
	// either branch to allocate a distinct slice header/backing array.
	if len(g.battlefield) > 0 {
		c.battlefield = g.battlefield[:len(g.battlefield):len(g.battlefield)]
		g.battlefieldSliceShared = true
		c.battlefieldSliceShared = true
	}
	g.battlefieldShared = true
	g.ownedPermanents = nil
	c.battlefieldShared = true

	// Deep copy exile zone.
	c.exile = make([]ExiledCard, len(g.exile))
	for i, ec := range g.exile {
		c.exile[i] = ExiledCard{
			Card:     ec.Card, // shared Card ref
			ExiledBy: ec.ExiledBy,
		}
	}

	// Deep copy stack.
	c.stack = cloneStack(g.stack)

	// Deep copy combat.
	c.combat = cloneCombat(g.combat)

	// Deep copy effect manager.
	c.effects = cloneEffectManager(g.effects)

	// Deep copy extra turns.
	if len(g.extraTurns) > 0 {
		c.extraTurns = make([]uuid.UUID, len(g.extraTurns))
		copy(c.extraTurns, g.extraTurns)
	}

	// Deep copy turn schedule.
	if g.schedule != nil {
		cs := newTurnSchedule()
		if len(g.schedule.Remaining) > 0 {
			cs.Remaining = make([]PhaseStep, len(g.schedule.Remaining))
			copy(cs.Remaining, g.schedule.Remaining)
		}
		maps.Copy(cs.SkipNextStep, g.schedule.SkipNextStep)
		maps.Copy(cs.SkipNextTurnFor, g.schedule.SkipNextTurnFor)
		c.schedule = cs
	}

	// Deep copy resolving targets.
	if len(g.resolvingTargets) > 0 {
		c.resolvingTargets = make([]uuid.UUID, len(g.resolvingTargets))
		copy(c.resolvingTargets, g.resolvingTargets)
	}

	// Deep copy pending triggers (share ability/event refs, they're read-only during search).
	if len(g.pendingTriggers) > 0 {
		c.pendingTriggers = make([]*pendingTrigger, len(g.pendingTriggers))
		for i, pt := range g.pendingTriggers {
			clone := *pt
			c.pendingTriggers[i] = &clone
		}
	}

	// Deep copy delayed triggers (share Effect refs).
	if len(g.delayedTriggers) > 0 {
		c.delayedTriggers = make([]*DelayedTrigger, len(g.delayedTriggers))
		for i, dt := range g.delayedTriggers {
			clone := *dt
			if len(dt.Effects) > 0 {
				clone.Effects = make([]Effect, len(dt.Effects))
				copy(clone.Effects, dt.Effects)
			}
			c.delayedTriggers[i] = &clone
		}
	}

	// Deep copy coin flip results.
	if len(g.coinFlipResults) > 0 {
		c.coinFlipResults = make([]bool, len(g.coinFlipResults))
		copy(c.coinFlipResults, g.coinFlipResults)
	}

	// Deep copy UUID-keyed maps.
	c.damageDealtBy = cloneNestedUUIDMap(g.damageDealtBy)
	c.damageTakenThisTurn = cloneUUIDIntMap(g.damageTakenThisTurn)
	c.artifactDamageTakenThisTurn = cloneUUIDIntMap(g.artifactDamageTakenThisTurn)
	c.combatDamageThisStep = make(map[uuid.UUID]map[uuid.UUID]int, len(g.combatDamageThisStep))
	for k, inner := range g.combatDamageThisStep {
		c.combatDamageThisStep[k] = cloneUUIDIntMap(inner)
	}
	c.combatDamageSourcesThisStep = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int, len(g.combatDamageSourcesThisStep))
	for k, byRecip := range g.combatDamageSourcesThisStep {
		dst := make(map[uuid.UUID]map[uuid.UUID]int, len(byRecip))
		for r, bySrc := range byRecip {
			dst[r] = cloneUUIDIntMap(bySrc)
		}
		c.combatDamageSourcesThisStep[k] = dst
	}
	c.attackedThisTurn = cloneUUIDBoolMap(g.attackedThisTurn)
	c.instantsCastThisTurn = cloneUUIDIntMap(g.instantsCastThisTurn)
	c.sorceriesCastThisTurn = cloneUUIDIntMap(g.sorceriesCastThisTurn)
	c.timesTargetedThisTurn = cloneUUIDMap(g.timesTargetedThisTurn)
	c.discardCountThisTurn = cloneUUIDMap(g.discardCountThisTurn)
	c.lifeGainedThisTurn = cloneUUIDMap(g.lifeGainedThisTurn)
	c.permDamageReceivedThisTurn = cloneUUIDMap(g.permDamageReceivedThisTurn)
	c.attackedOrBlockedThisTurn = cloneUUIDMap(g.attackedOrBlockedThisTurn)
	c.playerCastSpellThisTurn = cloneUUIDMap(g.playerCastSpellThisTurn)
	c.playerAttackedThisTurn = cloneUUIDMap(g.playerAttackedThisTurn)
	c.cardsDrawnThisTurn = cloneUUIDMap(g.cardsDrawnThisTurn)
	c.cardsLeftGraveyardThisTurn = cloneUUIDMap(g.cardsLeftGraveyardThisTurn)
	c.exileZoneChangesPending = cloneUUIDMap(g.exileZoneChangesPending)
	c.duelLandsPlayed = cloneUUIDMap(g.duelLandsPlayed)
	c.duelAttackersDeclared = cloneUUIDMap(g.duelAttackersDeclared)
	c.duelCreatureDeaths = cloneUUIDMap(g.duelCreatureDeaths)
	c.duelNonCombatDamage = cloneUUIDMap(g.duelNonCombatDamage)
	c.duelSpellsCastByColor = cloneUUIDColorMap(g.duelSpellsCastByColor)
	c.duelSpellsCastByType = cloneUUIDCardTypeMap(g.duelSpellsCastByType)
	c.blockedThisTurn = cloneBlockedThisTurn(g.blockedThisTurn)
	c.extraLandPlaysThisTurn = cloneUUIDMap(g.extraLandPlaysThisTurn)
	c.optionalCostPaid = cloneUUIDMap(g.optionalCostPaid)
	c.customState = cloneCustomState(g.customState)

	// Deep copy cast-from-exile permissions and exile-instead-of-graveyard tags.
	if len(g.castFromExilePermissions) > 0 {
		c.castFromExilePermissions = make([]CastableFromExilePermission, len(g.castFromExilePermissions))
		copy(c.castFromExilePermissions, g.castFromExilePermissions)
	}
	c.exileInsteadCards = make(map[uuid.UUID]uuid.UUID, len(g.exileInsteadCards))
	maps.Copy(c.exileInsteadCards, g.exileInsteadCards)
	c.armedStateTriggers = cloneStateTriggerMap(g.armedStateTriggers)

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
	bp.hand = cloneCardSlice(p.Hand())
	bp.graveyard = cloneCardSlice(p.Graveyard())
	bp.library = cloneCardSlice(p.Library())
	bp.ante = cloneCardSlice(p.Ante())
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
	// Copy card slices (sharing Card refs).
	clone.hand = cloneCardSlice(bp.hand)
	clone.graveyard = cloneCardSlice(bp.graveyard)
	clone.library = cloneCardSlice(bp.library)
	clone.ante = cloneCardSlice(bp.ante)
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
		Card:                src.Card, // shared
		Controller:          src.Controller,
		Tapped:              src.Tapped,
		PhasedOut:           src.PhasedOut,
		Damage:              src.Damage,
		AttachedTo:          src.AttachedTo,
		IsToken:             src.IsToken,
		FaceDown:            src.FaceDown,
		ChosenColor:         src.ChosenColor,
		ChosenPlayer:        src.ChosenPlayer,
		ChosenSubtype:       src.ChosenSubtype,
		ControlledPermanent: src.ControlledPermanent,
		TurnControlGained:   src.TurnControlGained,
		StoredValue:         src.StoredValue,
		CreatedBy:           src.CreatedBy,
		powerBonus:          src.powerBonus,
		toughBonus:          src.toughBonus,
		Counters:            src.Counters, // fixed-size array: value copy
		baseAttrs:           src.baseAttrs,
		grantedAttrs:        src.grantedAttrs,
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
	clone := &Stack{}
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
		ID:            obj.ID,
		Card:          obj.Card, // shared Card ref
		Controller:    obj.Controller,
		SourceID:      obj.SourceID,
		IsAbility:     obj.IsAbility,
		XValue:        obj.XValue,
		ModeChoice:    obj.ModeChoice,
		EventAmount:   obj.EventAmount,
		EventSourceID: obj.EventSourceID,
		IsCopy:        obj.IsCopy,
		CastZone:      obj.CastZone,
		CastContext:   obj.CastContext,
	}
	if len(obj.ModalTargets) > 0 {
		clone.ModalTargets = make([][]uuid.UUID, len(obj.ModalTargets))
		for i, t := range obj.ModalTargets {
			if len(t) > 0 {
				cp := make([]uuid.UUID, len(t))
				copy(cp, t)
				clone.ModalTargets[i] = cp
			}
		}
	}
	// Share Effect interface refs.
	if len(obj.Effects) > 0 {
		clone.Effects = make([]Effect, len(obj.Effects))
		copy(clone.Effects, obj.Effects)
	}
	// Copy targets.
	if len(obj.Targets) > 0 {
		clone.Targets = make([]uuid.UUID, len(obj.Targets))
		copy(clone.Targets, obj.Targets)
	}
	if len(obj.DamageDistribution) > 0 {
		clone.DamageDistribution = make(map[uuid.UUID]int, len(obj.DamageDistribution))
		maps.Copy(clone.DamageDistribution, obj.DamageDistribution)
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
		Attackers:     cloneUUIDBoolMap(c.Attackers),
		FirstStruck:   cloneUUIDBoolMap(c.FirstStruck),
		AttackedAlone: c.AttackedAlone,
		BlockedAlone:  c.BlockedAlone,
	}
	// Deep copy groups.
	if len(c.Groups) > 0 {
		clone.Groups = make([]*CombatGroup, len(c.Groups))
		for i, g := range c.Groups {
			cg := &CombatGroup{
				AttackerID: g.AttackerID,
				DefenderID: g.DefenderID,
				Blocked:    g.Blocked,
			}
			if len(g.BlockerIDs) > 0 {
				cg.BlockerIDs = make([]uuid.UUID, len(g.BlockerIDs))
				copy(cg.BlockerIDs, g.BlockerIDs)
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
	// Copy continuous effects slice (shared interface values — they receive *Game as parameter).
	if len(em.effects) > 0 {
		clone.effects = make([]ContinuousEffect, len(em.effects))
		copy(clone.effects, em.effects)
	}
	// Deep copy replacement effects (they hold mutable state).
	clone.replacements = cloneReplacementSlice(em.replacements)
	clone.cycleReplacements = cloneReplacementSlice(em.cycleReplacements)
	// Clone subsystems.
	clone.Rules = cloneGameRules(em.Rules)
	clone.Damage = cloneDamageSystem(em.Damage)
	clone.Damage.SetEffectManager(clone)
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
	clone.SpellCostReductions = cloneColorIntMap(gr.SpellCostReductions)
	clone.SpellTypeCostReductions = cloneCardTypeIntMap(gr.SpellTypeCostReductions)
	clone.sanctuaryActive = cloneUUIDBoolMap(gr.sanctuaryActive)
	clone.lichActive = cloneUUIDUUIDMap(gr.lichActive)
	clone.skipNextDraw = cloneUUIDBoolMap(gr.skipNextDraw)
	clone.channelActive = cloneUUIDBoolMap(gr.channelActive)
	clone.minimumLife = cloneUUIDBoolMap(gr.minimumLife)
	clone.maxHandSize = cloneUUIDIntMap(gr.maxHandSize)
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

// cloneDamageSystem deep copies the DamageSystem.
func cloneDamageSystem(ds *DamageSystem) *DamageSystem {
	clone := &DamageSystem{
		damageReflection: make(map[uuid.UUID]damageReflectionEntry, len(ds.damageReflection)),
	}
	maps.Copy(clone.damageReflection, ds.damageReflection)
	return clone
}

// --- Map clone helpers ---

// cloneCardSlice shares the backing array with the source but forces cap == len
// so any subsequent append on either side allocates a new backing array.
// Mutation paths that would otherwise touch shared memory (middle-element
// removal in RemoveFromHand/Graveyard/Ante, ShuffleLibrary) are required to
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

func cloneStateTriggerMap(src map[stateTriggerKey]bool) map[stateTriggerKey]bool {
	if src == nil {
		return make(map[stateTriggerKey]bool)
	}
	dst := make(map[stateTriggerKey]bool, len(src))
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
