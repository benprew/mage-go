package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// layerPostPT is not one of the CR 613.1 layers but the pass CR 613.11 asks
// for: continuous effects that modify the game rules rather than an object's
// characteristics are applied after all other continuous effects. Meekstone's
// "power 3 or greater" reads the power the layers produced, so in layer 6 it
// would still see a creature's power before the layer 7 modifications.
const layerPostPT Layer = LayerPT + 1

// ContinuousEffect represents an ongoing effect on the game.
type ContinuousEffect interface {
	Text() string
	Properties() EffectProperties
	Apply(*EffectContext) error
	GetLayer() Layer
	GetDuration() Duration
	IsActive(g *Game) bool
	SourceID() uuid.UUID
	SetSourceID(uuid.UUID)
}

// deepCloneableEffect is implemented by continuous effects that carry mutable
// per-game state. Game.Clone() shares effect interface values by default (they
// are otherwise stateless), so a stateful effect must deep-copy itself here —
// otherwise a throwaway AI search clone would corrupt the real effect through
// the shared pointer.
type deepCloneableEffect interface {
	cloneEffect() ContinuousEffect
}

type printedAbilityContinuousEffect struct {
	ContinuousEffect
}

func (e *printedAbilityContinuousEffect) IsActive(g *Game) bool {
	source := g.FindPermanent(e.SourceID())
	if source != nil && source.printedAbilitiesSuppressed {
		return false
	}
	return e.ContinuousEffect.IsActive(g)
}

func (e *printedAbilityContinuousEffect) cloneEffect() ContinuousEffect {
	inner := e.ContinuousEffect
	if cloneable, ok := inner.(deepCloneableEffect); ok {
		inner = cloneable.cloneEffect()
	}
	return &printedAbilityContinuousEffect{ContinuousEffect: inner}
}

// effectSource provides SourceID/SetSourceID to all continuous effects.
type effectSource struct {
	sourceID uuid.UUID
}

func (s *effectSource) SourceID() uuid.UUID      { return s.sourceID }
func (s *effectSource) SetSourceID(id uuid.UUID) { s.sourceID = id }

// ---------------------------------------------------------------------------
// Compositional continuous effect primitives
// ---------------------------------------------------------------------------

// ActiveCondition determines when a continuous effect is active.
type ActiveCondition func(g *Game, sourceID uuid.UUID) bool

// SourceAttached is active while the source is on the battlefield and attached to another permanent.
var SourceAttached ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && src.IsAttached()
}

// SourceUntapped is active while the source is on the battlefield and untapped.
var SourceUntapped ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && !src.Tapped
}

// SourceTapped is active while the source is on the battlefield and tapped.
var SourceTapped ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && src.Tapped
}

// WithSourceCondition bridges the existing SourceCondition type to ActiveCondition.
func WithSourceCondition(cond SourceCondition) ActiveCondition {
	return func(g *Game, sourceID uuid.UUID) bool {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return false
		}
		return cond(src, g)
	}
}

// ContinuousApplyFunc is the apply function for a functional continuous effect.
type ContinuousApplyFunc func(g *Game, sourceID uuid.UUID) error

// funcContinuousEffect implements ContinuousEffect using a function and metadata.
type funcContinuousEffect struct {
	layer    Layer
	duration Duration
	apply    ContinuousApplyFunc
	active   ActiveCondition // nil means SourceOnBattlefield
	effectSource
}

// FuncContinuousEffect creates a ContinuousEffect from a layer, duration, and apply function.
// By default, it's active while the source permanent is on the battlefield.
// Pass an optional ActiveCondition to customize when it's active.
func FuncContinuousEffect(layer Layer, duration Duration, apply ContinuousApplyFunc, condition ...ActiveCondition) ContinuousEffect {
	var cond ActiveCondition
	if len(condition) > 0 {
		cond = condition[0]
	}
	return &funcContinuousEffect{
		layer:    layer,
		duration: duration,
		apply:    apply,
		active:   cond,
	}
}

func (e *funcContinuousEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *funcContinuousEffect) Text() string                 { return "" }
func (e *funcContinuousEffect) GetLayer() Layer              { return e.layer }
func (e *funcContinuousEffect) GetDuration() Duration        { return e.duration }

func (e *funcContinuousEffect) IsActive(g *Game) bool {
	if e.active != nil {
		return e.active(g, e.sourceID)
	}
	// EndOfTurn/EndOfCombat effects are time-limited and don't require their source
	// on the battlefield (e.g. effects from resolved instants/sorceries).
	if e.duration == EndOfTurn || e.duration == EndOfCombat {
		return true
	}
	return g.FindPermanent(e.sourceID) != nil
}

func (e *funcContinuousEffect) Apply(ctx *EffectContext) error {
	return e.apply(ctx.Game, e.sourceID)
}

// AttachedApplyFunc receives both the source and the attached target.
type AttachedApplyFunc func(g *Game, source, target *Permanent) error

// attachedEffect implements ContinuousEffect for aura/equipment patterns.
type attachedEffect struct {
	layer Layer
	apply AttachedApplyFunc
	effectSource
}

// AttachedEffect creates a ContinuousEffect that applies while the source is
// attached to another permanent. The apply function receives both source and target.
func AttachedEffect(layer Layer, apply AttachedApplyFunc) ContinuousEffect {
	return &attachedEffect{
		layer: layer,
		apply: apply,
	}
}

func (e *attachedEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *attachedEffect) Text() string                 { return "" }
func (e *attachedEffect) GetLayer() Layer              { return e.layer }
func (e *attachedEffect) GetDuration() Duration        { return WhileOnBattlefield }

func (e *attachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *attachedEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.MutablePermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	return e.apply(g, src, target)
}

// TargetApplyFunc receives the targeted permanent.
type TargetApplyFunc func(g *Game, target *Permanent) error

// targetEffect implements ContinuousEffect for effects targeting a specific permanent.
type targetEffect struct {
	layer         Layer
	duration      Duration
	targetID      uuid.UUID
	apply         TargetApplyFunc
	active        ActiveCondition
	expired       bool
	tapMaintained bool
	effectSource
}

// TargetEffect creates a ContinuousEffect that applies to a specific permanent by ID.
// Active while the target permanent exists on the battlefield.
func TargetEffect(layer Layer, duration Duration, targetID uuid.UUID, apply TargetApplyFunc) ContinuousEffect {
	return TargetEffectWhen(layer, duration, targetID, apply, nil)
}

// TargetEffectWhen creates a target effect with an additional source-based
// active condition, such as SourceTapped.
func TargetEffectWhen(layer Layer, duration Duration, targetID uuid.UUID, apply TargetApplyFunc, condition ActiveCondition) ContinuousEffect {
	return &targetEffect{
		layer:    layer,
		duration: duration,
		targetID: targetID,
		apply:    apply,
		active:   condition,
	}
}

// TapMaintainedTargetEffect creates a target effect whose source keeps it active
// by remaining tapped (Tawnos's Weaponry, Phyrexian Gremlins, Willow Satyr, ...).
// condition must encode that dependency (e.g. SourceTapped). The untap step reads
// the tap-maintained flag: once the target leaves the battlefield the source
// untaps automatically instead of prompting "you may choose not to untap," since
// staying tapped no longer maintains anything.
func TapMaintainedTargetEffect(layer Layer, duration Duration, targetID uuid.UUID, apply TargetApplyFunc, condition ActiveCondition) ContinuousEffect {
	return &targetEffect{
		layer:         layer,
		duration:      duration,
		targetID:      targetID,
		apply:         apply,
		active:        condition,
		tapMaintained: true,
	}
}

func (e *targetEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *targetEffect) Text() string                 { return "" }
func (e *targetEffect) GetLayer() Layer              { return e.layer }
func (e *targetEffect) GetDuration() Duration        { return e.duration }

// cloneEffect deep-copies the effect so its mutable expired latch is isolated
// from the original when the game is cloned for AI search.
func (e *targetEffect) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}

func (e *targetEffect) IsActive(g *Game) bool {
	if g.FindPermanent(e.targetID) == nil {
		if g.FindPermanentIncludingPhased(e.targetID) != nil {
			return false
		}
		e.expired = true
		return false
	}
	if e.expired {
		return false
	}
	if e.active != nil {
		if !e.active(g, e.sourceID) {
			e.expired = true
			return false
		}
	}
	return true
}

func (e *targetEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	target := g.MutablePermanent(e.targetID)
	if target == nil {
		return nil
	}
	return e.apply(g, target)
}

// EffectManager manages and applies continuous effects.
type EffectManager struct {
	effects                    []ContinuousEffect
	attrDeltas                 map[uuid.UUID]map[Attr]int       // deltas accumulated during Apply(); written to perm.grantedAttrs
	blockPairRestrictions      map[uuid.UUID]map[uuid.UUID]bool // attacker -> set of blockers that can't block it; reset each Apply
	cantBeBlockedExceptByRules map[uuid.UUID][]PermanentFilter  // attacker -> conjunction of filters blockers must match
	canBlockOnlyRules          map[uuid.UUID][]PermanentFilter  // blocker -> conjunction of filters attackers must match
	minBlockers                map[uuid.UUID]int                // attacker -> minimum number of blockers required
	attackCosts                map[uuid.UUID][]Cost             // creature -> costs its controller must pay to declare it as an attacker (CR 508.1e)
	replacements               []ReplacementEffect              // persistent: one-shot, turn-scoped, while-on-battlefield
	cycleReplacements          []ReplacementEffect              // cleared each Apply() cycle, re-registered by continuous effects
	Rules                      *GameRules
}

func NewEffectManager() *EffectManager {
	em := &EffectManager{
		attrDeltas: make(map[uuid.UUID]map[Attr]int),
		Rules:      NewGameRules(),
	}
	return em
}

// GrantAttr records that the given attr is added to the given permanent by a
// continuous effect. Called during Apply() in layer/timestamp order; the result
// is written to perm.grantedAttrs at the end of the cycle.
//
// Ability add/remove is modeled as last-writer-wins, not an additive counter
// (CR 613.9): applying "has X" and "loses X" to the same object in timestamp
// order means the effect generated last wins. Because effects are applied in
// em.effects insertion order (which tracks timestamp; see game.go AddPermanent),
// assigning (rather than incrementing) makes the latest grant/revoke win.
func (em *EffectManager) GrantAttr(permID uuid.UUID, a Attr) {
	if em.attrDeltas[permID] == nil {
		em.attrDeltas[permID] = make(map[Attr]int)
	}
	em.attrDeltas[permID][a] = 1
}

// RevokeAttr records that the given attr is removed from the given permanent by
// a continuous effect (an ability-removing effect such as Hammerheim's "loses
// all landwalk"). Per CR 613.9 / 702.14e a removal is not a decrement: it makes
// the ability absent regardless of how many effects granted it (multiple
// instances of the same landwalk are redundant, not additive), unless a later
// grant re-adds it. See GrantAttr for the last-writer-wins model.
func (em *EffectManager) RevokeAttr(permID uuid.UUID, a Attr) {
	if em.attrDeltas[permID] == nil {
		em.attrDeltas[permID] = make(map[Attr]int)
	}
	em.attrDeltas[permID][a] = -1
}

// PreventBlockPair records that the given blocker cannot block the given attacker.
// Reset each Apply cycle; set by continuous effects (e.g. Argothian Pixies).
func (em *EffectManager) PreventBlockPair(blockerID, attackerID uuid.UUID) {
	if em.blockPairRestrictions == nil {
		em.blockPairRestrictions = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	if em.blockPairRestrictions[attackerID] == nil {
		em.blockPairRestrictions[attackerID] = make(map[uuid.UUID]bool)
	}
	em.blockPairRestrictions[attackerID][blockerID] = true
}

// IsBlockPrevented returns true if the blocker is prevented from blocking the attacker.
func (em *EffectManager) IsBlockPrevented(blockerID, attackerID uuid.UUID) bool {
	if em.blockPairRestrictions == nil {
		return false
	}
	return em.blockPairRestrictions[attackerID][blockerID]
}

func (em *EffectManager) Add(e ContinuousEffect) {
	em.effects = append(em.effects, e)
}

// SourceTapMaintainedStatus inspects the tap-maintained continuous effects the
// given source keeps active by remaining tapped. hasEffect reports whether the
// source has any such effect at all; targetLives reports whether at least one of
// their targets is still on the battlefield. The untap step auto-untaps a source
// with hasEffect && !targetLives (the maintained target is gone), while a source
// with no tap-maintained effects (storage lands, Tawnos's Coffin) is left to the
// normal "may choose not to untap" prompt.
func (em *EffectManager) SourceTapMaintainedStatus(g *Game, sourceID uuid.UUID) (hasEffect, targetLives bool) {
	for _, e := range em.effects {
		switch te := e.(type) {
		case *targetEffect:
			if !te.tapMaintained || te.SourceID() != sourceID {
				continue
			}
			hasEffect = true
			if g.FindPermanent(te.targetID) != nil {
				targetLives = true
			}
		case *controlContinuousEffect:
			if !te.tapMaintained || te.SourceID() != sourceID {
				continue
			}
			hasEffect = true
			if g.FindPermanent(te.targetID) != nil {
				targetLives = true
			}
		}
	}
	return hasEffect, targetLives
}

func (em *EffectManager) Remove(sourceID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.SourceID() != sourceID {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
	em.RemoveReplacements(sourceID)
}

// RemoveEndOfTurn removes all effects with EndOfTurn duration.
func (em *EffectManager) RemoveEndOfTurn() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfTurn {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveEndOfCombat removes all effects with EndOfCombat duration.
func (em *EffectManager) RemoveEndOfCombat() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfCombat {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveUntilYourNextTurn removes all effects with UntilYourNextTurn duration
// whose source is controlled by the given player. Called at the beginning of
// each player's upkeep to expire "until your next upkeep" effects.
func (em *EffectManager) RemoveUntilYourNextTurn(g *Game, controllerID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() == UntilYourNextTurn {
			src := g.FindPermanent(e.SourceID())
			if src != nil && src.ControllerID() == controllerID {
				continue // remove this effect
			}
			// Source left battlefield — also remove
			if src == nil {
				continue
			}
		}
		filtered = append(filtered, e)
	}
	em.effects = filtered
}

// Apply resets computed bonuses and reapplies all active effects in layer order.
func (em *EffectManager) Apply(g *Game) {
	clear(em.attrDeltas)
	clear(em.blockPairRestrictions)
	em.resetCombatRestrictions()
	em.cycleReplacements = em.cycleReplacements[:0]
	em.Rules.ResetPerCycle()

	controlStateMayChange := em.hasControlLayerEffects()
	if !controlStateMayChange {
		for _, p := range g.zones.battlefield {
			if p.computedController != p.baseController {
				controlStateMayChange = true
				break
			}
		}
	}
	var controllersAtCycleStart map[uuid.UUID]uuid.UUID
	if controlStateMayChange {
		controllersAtCycleStart = make(map[uuid.UUID]uuid.UUID, len(g.zones.battlefield))
		for _, p := range g.zones.battlefield {
			controllersAtCycleStart[p.ID()] = p.ControllerID()
		}
	}

	// Reset granted runtime abilities, subtype overrides, and grantedAttrs from effects.
	for _, p := range g.zones.battlefield {
		controllerNeedsReset := p.computedController != p.baseController
		if controllerNeedsReset {
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.computedController = p.baseController
		}
		if p.FaceDown {
			// Face-down permanents keep their overrides and empty abilities
			continue
		}
		needsReset := len(p.SubTypeOverride) > 0 ||
			len(p.SubTypeAdditions) > 0 ||
			p.hasSubtypeFamilyEffects() ||
			p.printedAbilitiesSuppressed ||
			p.BasePTOverride != nil ||
			p.ColorOverride != nil ||
			p.powerBonus != 0 ||
			p.toughBonus != 0
		for _, v := range p.grantedAttrs {
			if v != 0 {
				needsReset = true
				break
			}
		}
		if !needsReset {
			for _, a := range p.RuntimeAbilities {
				if _, ok := a.(*grantedByEffect); ok {
					needsReset = true
					break
				}
			}
		}
		if !needsReset {
			continue
		}
		p = g.MutablePermanent(p.ID())
		if p == nil {
			continue
		}
		if p.printedAbilitiesSuppressed {
			existing := make(map[uuid.UUID]bool, len(p.RuntimeAbilities))
			for _, a := range p.RuntimeAbilities {
				existing[a.AbilityID()] = true
			}
			restored := make([]Ability, 0, len(p.Card.Abilities())+len(p.RuntimeAbilities))
			for _, a := range p.Card.Abilities() {
				if !existing[a.AbilityID()] {
					restored = append(restored, a)
				}
			}
			p.RuntimeAbilities = append(restored, p.RuntimeAbilities...)
			p.printedAbilitiesSuppressed = false
		}
		base := p.RuntimeAbilities[:0]
		for _, a := range p.RuntimeAbilities {
			if _, ok := a.(*grantedByEffect); !ok {
				base = append(base, a)
			}
		}
		p.RuntimeAbilities = base
		p.SubTypeOverride = nil
		p.SubTypeAdditions = nil
		p.resetSubtypeFamilyEffects()
		p.BasePTOverride = nil
		p.ColorOverride = nil
		// Reset grantedAttrs and P/T bonuses so each Apply() cycle starts fresh.
		// Assigning a zero array is a fast memcpy — no allocation.
		p.grantedAttrs = [NumAttrs]int8{}
		p.powerBonus = 0
		p.toughBonus = 0
	}

	// Remove effects whose source is no longer on the battlefield
	// EndOfTurn and EndOfCombat effects persist until cleanup regardless of source (e.g., Giant Growth, Jade Statue)
	// Indefinite effects persist as long as they self-report active (e.g., Sleight of Mind)
	active := em.effects[:0]
	for _, e := range em.effects {
		dur := e.GetDuration()
		if dur == EndOfTurn || dur == EndOfCombat || dur == Indefinite || g.FindPermanent(e.SourceID()) != nil {
			active = append(active, e)
		}
	}
	em.effects = active

	for _, effect := range em.effects {
		if effect.GetLayer() == LayerCopy && effect.IsActive(g) {
			_ = effect.Apply(&EffectContext{Game: g})
			em.syncAttrDeltas(g)
		}
	}
	em.applyControlLayer(g)

	// Apply the remaining layers in order. Layers 1 and 2 were applied above.
	for _, layer := range []Layer{LayerCopy, LayerControl, LayerType, LayerColor, LayerAbility, LayerPT, layerPostPT} {
		if layer == LayerCopy || layer == LayerControl {
			continue
		}
		for _, e := range em.effects {
			if e.GetLayer() == layer && e.IsActive(g) {
				_ = e.Apply(&EffectContext{Game: g})
				em.syncAttrDeltas(g)
			}
		}
	}

	// Sync mana conversions to all player mana pools
	em.Rules.SyncManaConversions(g.players)

	em.syncAttrDeltas(g)

	if controlStateMayChange {
		// Post-layer enforcement: AttrCantChangeControl reverts any control changes
		// applied during LayerControl. This runs after attrs are written so that
		// effects granted at LayerAbility (e.g. Guardian Beast) take effect.
		for _, p := range g.zones.battlefield {
			priorController := controllersAtCycleStart[p.ID()]
			if p.HasAttr(AttrCantChangeControl) && p.ControllerID() != priorController {
				p = g.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.computedController = priorController
			}
		}

		for _, p := range g.zones.battlefield {
			priorController := controllersAtCycleStart[p.ID()]
			if p.ControllerID() != priorController {
				p = g.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.turnControlGained = g.turns.Turn()
				if !p.HasAttr(AttrSummonSick) {
					p.GrantBaseAttr(AttrSummonSick)
				}
			}
		}
	}

	for _, p := range g.zones.battlefield {
		g.syncAbilityContext(p)
	}
}

// syncAttrDeltas exposes the latest attribute result after each continuous
// effect so later effects see characteristics established earlier in layer and
// timestamp order (CR 613.1, 613.7).
func (em *EffectManager) syncAttrDeltas(g *Game) {
	for permID, delta := range em.attrDeltas {
		perm := g.MutablePermanent(permID)
		if perm == nil {
			continue
		}
		for a, d := range delta {
			switch {
			case d > 0:
				perm.grantedAttrs[a] = 1
			case d < 0:
				perm.grantedAttrs[a] = -(perm.baseAttrs[a] + 1)
			}
		}
	}
}

func (em *EffectManager) applyControlLayer(g *Game) {
	// Returning before touching permanents preserves their copy-on-write sharing
	// in cloned search states, which keeps minimax clone allocations bounded.
	if !em.hasControlLayerEffects() {
		g.layer2Controllers = nil
		return
	}

	controllers := make(map[uuid.UUID]uuid.UUID, len(g.zones.battlefield))
	for _, permanent := range g.zones.battlefield {
		controllers[permanent.ID()] = permanent.baseController
	}

	for range len(g.zones.battlefield) + 1 {
		g.layer2Controllers = controllers
		for _, permanent := range g.zones.battlefield {
			permanent = g.MutablePermanent(permanent.ID())
			if permanent != nil {
				permanent.computedController = permanent.baseController
			}
		}
		for _, effect := range em.effects {
			if effect.GetLayer() != LayerControl || !controlEffectActive(effect, g, false) {
				continue
			}
			_ = effect.Apply(&EffectContext{Game: g})
		}
		next := make(map[uuid.UUID]uuid.UUID, len(g.zones.battlefield))
		stable := true
		for _, permanent := range g.zones.battlefield {
			next[permanent.ID()] = permanent.ControllerID()
			if next[permanent.ID()] != controllers[permanent.ID()] {
				stable = false
			}
		}
		controllers = next
		if stable {
			break
		}
	}

	g.layer2Controllers = controllers
	for _, permanent := range g.zones.battlefield {
		permanent = g.MutablePermanent(permanent.ID())
		if permanent != nil {
			permanent.computedController = permanent.baseController
		}
	}
	for _, effect := range em.effects {
		if effect.GetLayer() != LayerControl || !controlEffectActive(effect, g, true) {
			continue
		}
		_ = effect.Apply(&EffectContext{Game: g})
	}
	g.layer2Controllers = nil
}

func (em *EffectManager) hasControlLayerEffects() bool {
	for _, effect := range em.effects {
		if effect.GetLayer() == LayerControl {
			return true
		}
	}
	return false
}

func controlEffectActive(effect ContinuousEffect, g *Game, latchExpiration bool) bool {
	if control, ok := effect.(*controlContinuousEffect); ok {
		return control.isActive(g, latchExpiration)
	}
	return effect.IsActive(g)
}

// HasKeywordIgnoringSource reports whether the permanent with the given ID would
// have keyword kw if continuous effects originating from ignoreSourceID were not
// applied.
//
// Earthbind needs this: its enters-the-battlefield ability deals 2 damage to the
// enchanted creature "if [it] has flying," but Earthbind's own static ability
// removes flying from that same creature. By the time the trigger resolves the
// creature has already lost flying, so a plain HasKeyword check would never see
// it. CR 603.4 reads the condition against the creature's flying before Earthbind
// strips it — i.e. ignoring Earthbind's own effect.
func (g *Game) HasKeywordIgnoringSource(permID uuid.UUID, kw Keyword, ignoreSourceID uuid.UUID) bool {
	saved := g.effects.effects
	filtered := make([]ContinuousEffect, 0, len(saved))
	for _, e := range saved {
		if e.SourceID() != ignoreSourceID {
			filtered = append(filtered, e)
		}
	}
	g.effects.effects = filtered
	g.effects.Apply(g)

	has := false
	if p := g.FindPermanent(permID); p != nil {
		has = p.HasKeyword(kw)
	}

	g.effects.effects = saved
	g.effects.Apply(g)
	return has
}

// ---------------------------------------------------------------------------
// Replacement effect management
// ---------------------------------------------------------------------------

// AddReplacement adds a persistent replacement effect.
func (em *EffectManager) AddReplacement(r ReplacementEffect) {
	em.replacements = append(em.replacements, r)
}

// PrependReplacement adds a persistent replacement effect at the front of the list
// so it is checked before any existing replacements.
func (em *EffectManager) PrependReplacement(r ReplacementEffect) {
	em.replacements = append([]ReplacementEffect{r}, em.replacements...)
}

// AddCycleReplacement adds a replacement that is cleared each Apply() cycle.
// Used by continuous effects that re-register their replacements each cycle.
func (em *EffectManager) AddCycleReplacement(r ReplacementEffect) {
	em.cycleReplacements = append(em.cycleReplacements, r)
}

// RemoveReplacements removes all replacement effects with the given sourceID.
func (em *EffectManager) RemoveReplacements(sourceID uuid.UUID) {
	filtered := em.replacements[:0]
	for _, r := range em.replacements {
		if r.SourceID() != sourceID {
			filtered = append(filtered, r)
		}
	}
	em.replacements = filtered

	filtered2 := em.cycleReplacements[:0]
	for _, r := range em.cycleReplacements {
		if r.SourceID() != sourceID {
			filtered2 = append(filtered2, r)
		}
	}
	em.cycleReplacements = filtered2
}

// ApplyReplacements runs the replacement pipeline on an action.
// Returns nil if the action was fully prevented/replaced.
// Each replacement fires at most once per event to prevent infinite loops.
func (em *EffectManager) ApplyReplacements(action Action, g *Game) Action {
	if dca, ok := action.(*DamageToCreatureAction); ok {
		if perm := g.FindPermanent(dca.PermanentID()); perm != nil && perm.HasAttr(AttrDamageCantBePreventedOrRedirected) {
			return action
		}
	}
	applied := make(map[ReplacementEffect]bool)
	tryMatch := func(action Action, preventionOnly bool) (Action, ReplacementEffect, bool) {
		for _, list := range [2][]ReplacementEffect{em.replacements, em.cycleReplacements} {
			for _, r := range list {
				if applied[r] {
					continue
				}
				if isPreventionReplacement(r) != preventionOnly {
					continue
				}
				if !r.IsActive(g) {
					continue
				}
				if r.Matches(action, g) {
					return r.Replace(action, g), r, true
				}
			}
		}
		return action, nil, false
	}
	for {
		if action == nil {
			return nil
		}
		newAction, r, ok := tryMatch(action, false)
		if ok {
			applied[r] = true
			action = newAction
			continue
		}
		newAction, r, ok = tryMatch(action, true)
		if ok {
			applied[r] = true
			action = newAction
			continue
		}
		break
	}
	return action
}

// ClearReplacementsEndOfTurn removes all replacement effects with EndOfTurn duration.
func (em *EffectManager) ClearReplacementsEndOfTurn() {
	filtered := em.replacements[:0]
	for _, r := range em.replacements {
		if r.GetDuration() != EndOfTurn {
			filtered = append(filtered, r)
		}
	}
	em.replacements = filtered
}

// grantedByEffect is a marker wrapper to identify abilities granted by continuous effects.
type grantedByEffect struct {
	Ability
	intrinsicBasicLandMana bool
}

// WrapGrantedAbility wraps an ability as granted-by-effect so it is cleaned up
// and re-applied each continuous effect cycle. Use this when a continuous effect
// needs to grant a triggered or activated ability to a permanent.
func WrapGrantedAbility(a Ability) Ability {
	return &grantedByEffect{Ability: a}
}

func wrapIntrinsicBasicLandManaAbility(a Ability) Ability {
	return &grantedByEffect{Ability: a, intrinsicBasicLandMana: true}
}
