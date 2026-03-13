package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// Clone creates a deep copy of the game state for AI search.
// Card objects and Effect interfaces are shared (immutable during play).
// All mutable state (permanents, players, maps, slices) is deep-copied.
// Interactive callbacks (OnPriority, etc.) are nil'd in the clone.
// Players are wrapped in SearchPlayer for non-interactive choice defaults.
func (g *Game) Clone() *Game {
	c := &Game{
		Turn:                  g.Turn,
		Step:                  g.Step,
		ActivePlayer:          g.ActivePlayer,
		CurrentX:              g.CurrentX,
		CurrentMode:           g.CurrentMode,
		CurrentEventAmount:    g.CurrentEventAmount,
		ResolvingCard:         g.ResolvingCard, // Card ref shared
		LandsPlayedThisTurn:   g.LandsPlayedThisTurn,
		CreatureDeathsThisTurn: g.CreatureDeathsThisTurn,
		stopped:               g.stopped,
		resolvingCombatDamage: g.resolvingCombatDamage,
	}

	// Deep copy players, wrapping in SearchPlayer for non-interactive choices.
	c.Players = make([]Player, len(g.Players))
	for i, p := range g.Players {
		c.Players[i] = clonePlayer(p)
	}

	// Deep copy battlefield (permanents share Card refs).
	c.Battlefield = make([]*Permanent, len(g.Battlefield))
	for i, p := range g.Battlefield {
		c.Battlefield[i] = clonePermanent(p)
	}

	// Deep copy exile zone.
	c.Exile = make([]ExiledCard, len(g.Exile))
	for i, ec := range g.Exile {
		c.Exile[i] = ExiledCard{
			Card:     ec.Card, // shared Card ref
			ExiledBy: ec.ExiledBy,
			Owner:    ec.Owner,
			Counters: cloneCounterMap(ec.Counters),
		}
	}

	// Deep copy stack.
	c.Stack = cloneStack(g.Stack)

	// Deep copy combat.
	c.Combat = cloneCombat(g.Combat)

	// Deep copy effect manager.
	c.Effects = cloneEffectManager(g.Effects)

	// Deep copy extra turns.
	if len(g.ExtraTurns) > 0 {
		c.ExtraTurns = make([]uuid.UUID, len(g.ExtraTurns))
		copy(c.ExtraTurns, g.ExtraTurns)
	}

	// Deep copy resolving targets.
	if len(g.ResolvingTargets) > 0 {
		c.ResolvingTargets = make([]uuid.UUID, len(g.ResolvingTargets))
		copy(c.ResolvingTargets, g.ResolvingTargets)
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
	if len(g.CoinFlipResults) > 0 {
		c.CoinFlipResults = make([]bool, len(g.CoinFlipResults))
		copy(c.CoinFlipResults, g.CoinFlipResults)
	}

	// Deep copy UUID-keyed maps.
	c.DamageDealtBy = cloneNestedUUIDMap(g.DamageDealtBy)
	c.DamageTakenThisTurn = cloneUUIDIntMap(g.DamageTakenThisTurn)
	c.ArtifactDamageTakenThisTurn = cloneUUIDIntMap(g.ArtifactDamageTakenThisTurn)
	c.ArtifactManaOnly = cloneUUIDBoolMap(g.ArtifactManaOnly)
	c.CreatureManaOnly = cloneUUIDBoolMap(g.CreatureManaOnly)
	c.AttackedThisTurn = cloneUUIDBoolMap(g.AttackedThisTurn)
	c.InstantsCastThisTurn = cloneUUIDIntMap(g.InstantsCastThisTurn)
	c.BlockedThisTurn = cloneBlockedThisTurn(g.BlockedThisTurn)

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
		id:             p.PlayerID(),
		name:           p.Name(),
		life:           p.Life(),
		drewFromEmpty:  p.DrewFromEmpty(),
		manaPool:       NewManaPool(),
		poisonCounters: p.PoisonCounters(),
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
		for k, v := range p.ManaPool().ManaConversions {
			bp.manaPool.ManaConversions[k] = v
		}
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
		for k, v := range bp.manaPool.ManaConversions {
			clone.manaPool.ManaConversions[k] = v
		}
	}
	return clone
}

// clonePermanent creates a deep copy of a Permanent, sharing the Card ref.
func clonePermanent(p *Permanent) *Permanent {
	clone := &Permanent{
		Card:                p.Card, // shared
		Controller:          p.Controller,
		Tapped:              p.Tapped,
		PhasedOut:           p.PhasedOut,
		Damage:              p.Damage,
		AttachedTo:          p.AttachedTo,
		FaceDown:            p.FaceDown,
		ChosenColor:         p.ChosenColor,
		ChosenPlayer:        p.ChosenPlayer,
		ControlledPermanent: p.ControlledPermanent,
		TurnControlGained:   p.TurnControlGained,
		StoredValue:         p.StoredValue,
		CreatedBy:           p.CreatedBy,
		BasePTOverride:      p.BasePTOverride, // immutable *[2]int pointer (set once)
		ColorOverride:       p.ColorOverride,  // same — set once by continuous effect init
		powerBonus:          p.powerBonus,
		toughBonus:          p.toughBonus,
	}
	// Deep copy counters.
	clone.Counters = cloneCounterMap(p.Counters)
	// Deep copy attachments.
	if len(p.Attachments) > 0 {
		clone.Attachments = make([]uuid.UUID, len(p.Attachments))
		copy(clone.Attachments, p.Attachments)
	}
	// Deep copy runtime abilities (share Ability interface refs).
	if len(p.RuntimeAbilities) > 0 {
		clone.RuntimeAbilities = make([]Ability, len(p.RuntimeAbilities))
		copy(clone.RuntimeAbilities, p.RuntimeAbilities)
	}
	// Deep copy subtype override.
	if len(p.SubTypeOverride) > 0 {
		clone.SubTypeOverride = make([]string, len(p.SubTypeOverride))
		copy(clone.SubTypeOverride, p.SubTypeOverride)
	}
	// Deep copy BasePTOverride if non-nil (it's a *[2]int).
	if p.BasePTOverride != nil {
		v := *p.BasePTOverride
		clone.BasePTOverride = &v
	}
	// Deep copy ColorOverride if non-nil.
	if p.ColorOverride != nil {
		colors := make([]Color, len(*p.ColorOverride))
		copy(colors, *p.ColorOverride)
		clone.ColorOverride = &colors
	}
	// Deep copy attr maps.
	clone.baseAttrs = cloneAttrMap(p.baseAttrs)
	clone.grantedAttrs = cloneAttrMap(p.grantedAttrs)
	return clone
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
		ID:          obj.ID,
		Card:        obj.Card, // shared Card ref
		Controller:  obj.Controller,
		SourceID:    obj.SourceID,
		IsAbility:   obj.IsAbility,
		XValue:      obj.XValue,
		ModeChoice:  obj.ModeChoice,
		EventAmount: obj.EventAmount,
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
	return clone
}

// cloneCombat deep copies the Combat struct.
func cloneCombat(c *Combat) *Combat {
	if c == nil {
		return nil
	}
	clone := &Combat{
		Attackers:   cloneUUIDBoolMap(c.Attackers),
		FirstStruck: cloneUUIDBoolMap(c.FirstStruck),
	}
	// Deep copy groups.
	if len(c.Groups) > 0 {
		clone.Groups = make([]*CombatGroup, len(c.Groups))
		for i, g := range c.Groups {
			cg := &CombatGroup{
				AttackerID: g.AttackerID,
				DefenderID: g.DefenderID,
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
	for k, v := range ds.damageReflection {
		clone.damageReflection[k] = v
	}
	return clone
}

// --- Map clone helpers ---

func cloneCardSlice(src []Card) []Card {
	if src == nil {
		return nil
	}
	dst := make([]Card, len(src))
	copy(dst, src) // Card refs are shared (immutable)
	return dst
}

func cloneCounterMap(src map[CounterType]int) map[CounterType]int {
	if src == nil {
		return make(map[CounterType]int)
	}
	dst := make(map[CounterType]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneAttrMap(src map[Attr]int) map[Attr]int {
	if src == nil {
		return make(map[Attr]int)
	}
	dst := make(map[Attr]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneUUIDBoolMap(src map[uuid.UUID]bool) map[uuid.UUID]bool {
	if src == nil {
		return make(map[uuid.UUID]bool)
	}
	dst := make(map[uuid.UUID]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneUUIDIntMap(src map[uuid.UUID]int) map[uuid.UUID]int {
	if src == nil {
		return make(map[uuid.UUID]int)
	}
	dst := make(map[uuid.UUID]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneUUIDUUIDMap(src map[uuid.UUID]uuid.UUID) map[uuid.UUID]uuid.UUID {
	if src == nil {
		return make(map[uuid.UUID]uuid.UUID)
	}
	dst := make(map[uuid.UUID]uuid.UUID, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneNestedUUIDMap(src map[uuid.UUID]map[uuid.UUID]bool) map[uuid.UUID]map[uuid.UUID]bool {
	if src == nil {
		return make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	dst := make(map[uuid.UUID]map[uuid.UUID]bool, len(src))
	for k, v := range src {
		inner := make(map[uuid.UUID]bool, len(v))
		for ik, iv := range v {
			inner[ik] = iv
		}
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

func cloneColorColorMap(src map[Color]Color) map[Color]Color {
	if src == nil {
		return make(map[Color]Color)
	}
	dst := make(map[Color]Color, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneColorIntMap(src map[Color]int) map[Color]int {
	if src == nil {
		return make(map[Color]int)
	}
	dst := make(map[Color]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneCardTypeIntMap(src map[CardType]int) map[CardType]int {
	if src == nil {
		return make(map[CardType]int)
	}
	dst := make(map[CardType]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneAttrBoolMap(src map[Attr]bool) map[Attr]bool {
	if src == nil {
		return make(map[Attr]bool)
	}
	dst := make(map[Attr]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
