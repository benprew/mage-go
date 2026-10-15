package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ContinuousEffect represents an ongoing effect on the game.
type ContinuousEffect interface {
	EffectMetadata
	ApplyContinuous(*EffectContext) error
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
	declaration *StaticAbilityHolder
	acquired    bool
	incarnation uuid.UUID
}

func (e *printedAbilityContinuousEffect) IsActive(g *Game) bool {
	source := g.FindPermanent(e.SourceID())
	if e.declaration != nil && (source == nil || source.incarnationID != e.incarnation) {
		return false
	}
	if continuation, ok := e.ContinuousEffect.(interface{ continuesAcrossLayers(*Game) bool }); ok && continuation.continuesAcrossLayers(g) {
		return true
	}
	if e.declaration != nil && !slices.ContainsFunc(source.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.declaration.AbilityID() }) {
		return false
	}
	if !e.acquired && source != nil && source.printedAbilitiesSuppressed {
		return false
	}
	return e.ContinuousEffect.IsActive(g)
}

func (e *printedAbilityContinuousEffect) GetSublayer() SublayerPT {
	return getEffectSublayer(e.ContinuousEffect)
}

func (e *printedAbilityContinuousEffect) DependsOn(other ContinuousEffect, g *Game) bool {
	if printed, ok := other.(*printedAbilityContinuousEffect); ok {
		other = printed.ContinuousEffect
	}
	if removal, ok := other.(*abilityRemovalContinuousEffect); ok {
		source := g.FindPermanent(e.SourceID())
		return source != nil && removal.matches(source, g)
	}
	if removal, ok := other.(*composedLayerEffect); ok && removal.removesAbilities() {
		probe := probeContinuousEffect(other, g)
		if probe != nil && e.IsActive(g) != e.IsActive(probe) {
			return true
		}
	}
	if e.GetLayer() == LayerType && other.GetLayer() == LayerType {
		probe := probeContinuousEffect(other, g)
		if probe != nil && e.IsActive(g) != e.IsActive(probe) {
			return true
		}
	}
	if detector, ok := e.ContinuousEffect.(DependencyDetector); ok {
		return detector.DependsOn(other, g)
	}
	return false
}

func (e *printedAbilityContinuousEffect) cloneEffect() ContinuousEffect {
	inner := e.ContinuousEffect
	if cloneable, ok := inner.(deepCloneableEffect); ok {
		inner = cloneable.cloneEffect()
	}
	clone := *e
	clone.ContinuousEffect = inner
	return &clone
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
	sublayer SublayerPT
	duration Duration
	apply    ContinuousApplyFunc
	active   ActiveCondition // nil means SourceOnBattlefield
	effectSource
}

// FuncContinuousEffect creates a ContinuousEffect from a layer, duration, and apply function.
// By default, it's active while the source permanent is on the battlefield.
// Pass an optional ActiveCondition to customize when it's active.
func FuncContinuousEffect(layer Layer, duration Duration, apply ContinuousApplyFunc, condition ...ActiveCondition) ContinuousEffect {
	return FuncContinuousEffectWithSublayer(layer, SublayerPT_Modify, duration, apply, condition...)
}

// FuncContinuousEffectWithSublayer creates a ContinuousEffect with an explicit Layer 7 sublayer.
func FuncContinuousEffectWithSublayer(layer Layer, sublayer SublayerPT, duration Duration, apply ContinuousApplyFunc, condition ...ActiveCondition) ContinuousEffect {
	var cond ActiveCondition
	if len(condition) > 0 {
		cond = condition[0]
	}
	return &funcContinuousEffect{
		layer:    layer,
		sublayer: sublayer,
		duration: duration,
		apply:    apply,
		active:   cond,
	}
}

func (e *funcContinuousEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *funcContinuousEffect) Text() string                 { return "" }
func (e *funcContinuousEffect) GetLayer() Layer              { return e.layer }
func (e *funcContinuousEffect) GetSublayer() SublayerPT {
	if e.sublayer == 0 {
		return SublayerPT_Modify
	}
	return e.sublayer
}
func (e *funcContinuousEffect) GetDuration() Duration { return e.duration }

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

func (e *funcContinuousEffect) ApplyContinuous(ctx *EffectContext) error {
	return e.apply(ctx.Game, e.sourceID)
}

// AttachedApplyFunc receives both the source and the attached target.
type AttachedApplyFunc func(g *Game, source, target *Permanent) error

// attachedEffect implements ContinuousEffect for aura/equipment patterns.
type attachedEffect struct {
	layer    Layer
	sublayer SublayerPT
	apply    AttachedApplyFunc
	effectSource
}

// AttachedEffect creates a ContinuousEffect that applies while the source is
// attached to the same permanent incarnation. The apply function receives both
// source and target.
func AttachedEffect(layer Layer, apply AttachedApplyFunc) ContinuousEffect {
	return &attachedEffect{
		layer:    layer,
		sublayer: SublayerPT_Modify,
		apply:    apply,
	}
}

func (e *attachedEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *attachedEffect) Text() string                 { return "" }
func (e *attachedEffect) GetLayer() Layer              { return e.layer }
func (e *attachedEffect) GetSublayer() SublayerPT {
	if e.sublayer == 0 {
		return SublayerPT_Modify
	}
	return e.sublayer
}
func (e *attachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *attachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return false
	}
	target := g.FindPermanent(src.AttachedTo)
	return target != nil && target.incarnationID == src.attachedIncarnation
}

func (e *attachedEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.MutablePermanent(src.AttachedTo)
	if target == nil || target.incarnationID != src.attachedIncarnation {
		return nil
	}
	return e.apply(g, src, target)
}

// TargetApplyFunc receives the targeted permanent.
type TargetApplyFunc func(g *Game, target *Permanent) error

// targetEffect implements ContinuousEffect for effects targeting a specific permanent.
type targetEffect struct {
	layer         Layer
	sublayer      SublayerPT
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

// TargetEffectWithSublayer creates a target effect with an explicit Layer 7 sublayer.
func TargetEffectWithSublayer(layer Layer, sublayer SublayerPT, duration Duration, targetID uuid.UUID, apply TargetApplyFunc) ContinuousEffect {
	return &targetEffect{
		layer:    layer,
		sublayer: sublayer,
		duration: duration,
		targetID: targetID,
		apply:    apply,
	}
}

// TargetEffectWhen creates a target effect with an additional source-based
// active condition, such as SourceTapped.
func TargetEffectWhen(layer Layer, duration Duration, targetID uuid.UUID, apply TargetApplyFunc, condition ActiveCondition) ContinuousEffect {
	return &targetEffect{
		layer:    layer,
		sublayer: SublayerPT_Modify,
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
		sublayer:      SublayerPT_Modify,
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
func (e *targetEffect) GetSublayer() SublayerPT {
	if e.sublayer == 0 {
		return SublayerPT_Modify
	}
	return e.sublayer
}
func (e *targetEffect) GetDuration() Duration { return e.duration }

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

func (e *targetEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	target := g.MutablePermanent(e.targetID)
	if target == nil {
		return nil
	}
	return e.apply(g, target)
}

// EffectManager manages and applies continuous effects.
type EffectManager struct {
	animationMembers           map[animationCycleKey][]capturedPermanent
	composedMembers            map[composedCycleKey]composedCycleSelection
	bandsWithOther             map[capturedPermanent][]PermanentFilter
	effects                    []ContinuousEffect
	attrDeltas                 map[uuid.UUID]map[Attr]int       // deltas accumulated during Apply(); written to perm.grantedAttrs
	blockPairRestrictions      map[uuid.UUID]map[uuid.UUID]bool // attacker -> set of blockers that can't block it; reset each Apply
	noCombatAssignmentRules    map[uuid.UUID]uuid.UUID
	blockAnyNumberRules        map[uuid.UUID]bool
	blockEachAttackerRules     map[uuid.UUID]int
	allCreaturesMustBlockRules map[uuid.UUID][]PermanentFilter
	cantBeBlockedExceptByRules map[uuid.UUID][]PermanentFilter // attacker -> conjunction of filters blockers must match
	canBlockOnlyRules          map[uuid.UUID][]PermanentFilter // blocker -> conjunction of filters attackers must match
	minBlockers                map[uuid.UUID]int               // attacker -> minimum number of blockers required
	attackCosts                map[uuid.UUID][]Cost            // creature -> costs its controller must pay to declare it as an attacker (CR 508.1e)
	replacements               []ReplacementEffect             // persistent: one-shot, turn-scoped, while-on-battlefield
	cycleReplacements          []ReplacementEffect             // cleared each Apply() cycle, re-registered by continuous effects
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
	if a == Banding {
		for subject := range em.bandsWithOther {
			if subject.id == permID {
				delete(em.bandsWithOther, subject)
			}
		}
	}
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

func (em *EffectManager) retimestampAttachment(sourceID uuid.UUID) {
	// Static abilities use the attachment's new timestamp after a move.
	ordered := make([]ContinuousEffect, 0, len(em.effects))
	moved := make([]ContinuousEffect, 0)
	for _, effect := range em.effects {
		if _, printed := effect.(*printedAbilityContinuousEffect); printed && effect.SourceID() == sourceID {
			moved = append(moved, effect)
		} else {
			ordered = append(ordered, effect)
		}
	}
	ordered = append(ordered, moved...)
	em.effects = ordered
}

// A later tap or phase-in must not start the old duration again.
func (em *EffectManager) expireSourceTapped(sourceID uuid.UUID) bool {
	changed := em.expireCapturedControlSource(sourceID, true)
	for _, effect := range em.effects {
		if e, ok := effect.(*capturedTappedUntapRule); ok && e.SourceID() == sourceID && !e.expired {
			e.expired = true
			changed = true
		}
		if e, ok := effect.(*composedLayerEffect); ok && e.sourceTapped && e.SourceID() == sourceID && !e.expired {
			e.expired = true
			changed = true
		}
	}
	return changed
}

func (em *EffectManager) expireSourcePresence(sourceID uuid.UUID) bool {
	changed := false
	for _, effect := range em.effects {
		if e, ok := effect.(*composedLayerEffect); ok && e.sourcePresent && e.SourceID() == sourceID && !e.expired {
			e.expired = true
			changed = true
		}
	}
	return changed
}

func (em *EffectManager) expireCapturedControlSource(sourceID uuid.UUID, tappedOnly bool) bool {
	changed := false
	for _, effect := range em.effects {
		if e, ok := effect.(*controlContinuousEffect); ok && e.sourceIncarnation != uuid.Nil && e.SourceID() == sourceID && !e.expired && (!tappedOnly || e.sourceTapped) {
			e.expired = true
			changed = true
		}
	}
	return changed
}

func (em *EffectManager) expireControlPowerSubject(id uuid.UUID) bool {
	changed := false
	for _, effect := range em.effects {
		control, ok := effect.(*controlContinuousEffect)
		if !ok || control.targetID != id || control.expired {
			continue
		}
		for _, condition := range control.conditions {
			if _, power := condition.(ControlTargetPowerLESource); power {
				control.expired = true
				changed = true
			}
		}
	}
	return changed
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
		independent, ok := e.(interface{ survivesSourceExit() bool })
		if e.SourceID() != sourceID || (ok && independent.survivesSourceExit()) {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
	em.removeReplacements(sourceID, true)
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
	replacements := em.replacements[:0]
	for _, r := range em.replacements {
		if r.GetDuration() != EndOfCombat {
			replacements = append(replacements, r)
		}
	}
	em.replacements = replacements
}

// RemoveUntilYourNextTurn uses a saved player when the effect supplies one.
// Other callers keep the source controller as their duration policy.
func (em *EffectManager) RemoveUntilYourNextTurn(g *Game, controllerID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() == UntilYourNextTurn {
			if captured, ok := e.(*composedLayerEffect); ok && captured.upkeepPlayer != uuid.Nil {
				if captured.endNextUpkeep {
					if captured.upkeepPlayer == controllerID {
						captured.upkeepReady = true
					}
					filtered = append(filtered, e)
				} else if captured.upkeepPlayer != controllerID {
					filtered = append(filtered, e)
				}
				continue
			}
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
	g.resetPrintedLandwalkTextLayers()
	g.resetPrintedProtectionTextLayers()
	g.resetPrintedStaticAbilityTextLayers()
	g.resetPrintedManaTapTextLayers()
	g.resetPrintedSpellCastTextLayers()
	g.resetPrintedUpkeepTextLayers()
	g.resetPrintedCombatPairTextLayers()
	g.resetPrintedLandTapTextLayers()
	g.resetPrintedActivatedTextLayers()
	// CR 613.6 keeps each animation selection for this layer cycle.
	em.animationMembers = make(map[animationCycleKey][]capturedPermanent)
	em.composedMembers = make(map[composedCycleKey]composedCycleSelection)
	clear(em.attrDeltas)
	clear(em.bandsWithOther)
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
			p = g.MutablePermanent(p.ID())
			if p != nil {
				p.resetComputedPT()
			}
			continue
		}
		needsReset := p.cardTypeOverride != nil || len(p.SubTypeOverride) > 0 ||
			len(p.SubTypeAdditions) > 0 ||
			p.hasSubtypeFamilyEffects() ||
			p.printedAbilitiesSuppressed || p.creatureEnchantRemoved ||
			p.ptSwitched ||
			p.cdaPower != nil ||
			p.cdaToughness != nil ||
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
					if revised := p.revisedPrintedTriggers[a.AbilityID()]; revised != nil {
						restored = append(restored, revised)
					} else if action := p.revisedPrintedActions[a.AbilityID()]; action != nil {
						restored = append(restored, action)
					} else if bonus := p.revisedPrintedManaBonuses[a.AbilityID()]; bonus != nil {
						restored = append(restored, bonus)
					} else if landwalk := p.revisedPrintedLandwalks[a.AbilityID()]; landwalk != nil {
						restored = append(restored, landwalk)
					} else if protection := p.revisedPrintedProtections[a.AbilityID()]; protection != nil {
						restored = append(restored, protection)
					} else {
						restored = append(restored, g.registeredPrintedStaticAbility(p, a))
					}
				}
			}
			p.RuntimeAbilities = append(restored, p.RuntimeAbilities...)
			p.printedAbilitiesSuppressed = false
		}
		// A clone can share this storage. Keep later grants in a separate slice.
		base := make([]Ability, 0, len(p.RuntimeAbilities))
		for _, a := range p.RuntimeAbilities {
			if _, ok := a.(*grantedByEffect); !ok {
				base = append(base, a)
			}
		}
		p.RuntimeAbilities = base
		p.creatureEnchantRemoved = false
		p.cardTypeOverride = nil
		p.SubTypeOverride = nil
		p.SubTypeAdditions = nil
		p.resetSubtypeFamilyEffects()
		p.BasePTOverride = nil
		p.ColorOverride = nil
		// Reset grantedAttrs and P/T bonuses so each Apply() cycle starts fresh.
		// Assigning a zero array is a fast memcpy — no allocation.
		p.grantedAttrs = [NumAttrs]int8{}
		p.resetComputedPT()
	}

	// Remove effects whose source is no longer on the battlefield
	// EndOfTurn and EndOfCombat effects persist until cleanup regardless of source (e.g., Giant Growth, Jade Statue)
	// Indefinite effects persist as long as they self-report active (e.g., Sleight of Mind)
	active := em.effects[:0]
	for _, e := range em.effects {
		dur := e.GetDuration()
		independent, ok := e.(interface{ survivesSourceExit() bool })
		if dur == EndOfTurn || dur == EndOfCombat || dur == Indefinite || (ok && independent.survivesSourceExit()) || g.FindPermanentIncludingPhased(e.SourceID()) != nil {
			active = append(active, e)
		}
	}
	em.effects = active

	for _, effect := range em.effects {
		if effect.GetLayer() == LayerCopy && effect.IsActive(g) {
			_ = effect.ApplyContinuous(&EffectContext{Game: g})
			em.syncAttrDeltas(g)
		}
	}
	em.applyControlLayer(g)

	// Apply the remaining layers in order. Layers 1 and 2 were applied above.
	for _, layer := range []Layer{LayerCopy, LayerControl, LayerText, LayerType, LayerColor, LayerAbility, LayerPT} {
		if layer == LayerCopy || layer == LayerControl {
			continue
		}
		if layer == LayerPT {
			// CR 613.4: Apply Layer 7 sublayers in order:
			// 7a: CDAs (CR 613.4a)
			// 7b: Setting base P/T (CR 613.4b)
			// 7c: Modifications without setting (CR 613.4c)
			// 7d: Switching P/T (CR 613.4d)
			for _, sub := range []SublayerPT{SublayerPT_CDA, SublayerPT_Set, SublayerPT_Modify, SublayerPT_Switch} {
				var subEffects []ContinuousEffect
				for _, e := range em.effects {
					if e.GetLayer() == LayerPT && getEffectSublayer(e) == sub && e.IsActive(g) {
						subEffects = append(subEffects, e)
					}
				}
				subEffects = orderEffectsByDependency(subEffects, g)
				for _, e := range subEffects {
					_ = e.ApplyContinuous(&EffectContext{Game: g})
					em.syncAttrDeltas(g)
				}
			}
			continue
		}
		var layerEffects []ContinuousEffect
		for _, e := range em.effects {
			if e.GetLayer() == layer {
				layerEffects = append(layerEffects, e)
			}
		}
		// A granted ability can make an inactive effect active in this layer.
		for len(layerEffects) > 0 {
			var activeEffects []ContinuousEffect
			var activeIndices []int
			for i, effect := range layerEffects {
				if effect.IsActive(g) {
					activeEffects = append(activeEffects, effect)
					activeIndices = append(activeIndices, i)
				}
			}
			if len(activeEffects) == 0 {
				break
			}
			next := activeIndices[effectDependencyOrder(activeEffects, g)[0]]
			e := layerEffects[next]
			layerEffects = slices.Delete(layerEffects, next, next+1)
			if !e.IsActive(g) {
				continue
			}
			_ = e.ApplyContinuous(&EffectContext{Game: g})
			em.syncAttrDeltas(g)
		}

		if layer == LayerType {
			// Printed land types also supply mana abilities under CR 305.6.
			for _, permanent := range g.zones.battlefield {
				if permanent.PhasedOut || permanent.FaceDown || !permanent.HasType(TypeLand) ||
					!slices.ContainsFunc(permanent.computedSubtypes(), isBasicLandSubtype) {
					continue
				}
				if permanent = g.MutablePermanent(permanent.ID()); permanent != nil {
					refreshIntrinsicBasicLandManaAbilities(permanent)
				}
			}
		}
	}

	// Sync mana conversions to all player mana pools
	em.Rules.SyncManaConversions(g.players)

	em.syncAttrDeltas(g)

	if controlStateMayChange {
		expired := false
		for _, effect := range em.effects {
			if control, ok := effect.(*controlContinuousEffect); ok {
				expired = control.expirePowerEndpoint(g) || expired
			}
		}
		if expired {
			// A duration uses completed layers, before control changes are observed.
			for id, controller := range controllersAtCycleStart {
				if p := g.MutablePermanent(id); p != nil {
					p.computedController = controller
				}
			}
			em.Apply(g)
			return
		}
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
				if p.controlGainedTurns == nil {
					p.controlGainedTurns = make(map[uuid.UUID]int)
				}
				p.controlGainedTurns[p.ControllerID()] = g.turns.Turn()
				if !p.HasAttr(AttrSummonSick) {
					p.GrantBaseAttr(AttrSummonSick)
				}
				g.combat.RemoveFromCombat(p.ID())
			}
		}
	}

	for _, p := range g.zones.battlefield {
		g.syncAbilityContext(p)
	}
	g.syncManaSpendingPermissions()
	g.syncHandVisibility()
	g.syncLibraryTopVisibility()
	g.CheckStateTriggers()
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
			_ = effect.ApplyContinuous(&EffectContext{Game: g})
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
		_ = effect.ApplyContinuous(&EffectContext{Game: g})
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
		return control.isActive(g, latchExpiration, true)
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
	em.removeReplacements(sourceID, false)
}

func (em *EffectManager) removeReplacements(sourceID uuid.UUID, sourceExit bool) {
	keep := func(r ReplacementEffect) bool {
		independent, ok := r.(interface{ survivesSourceExit() bool })
		return r.SourceID() != sourceID || (sourceExit && ok && independent.survivesSourceExit())
	}
	filtered := em.replacements[:0]
	for _, r := range em.replacements {
		if keep(r) {
			filtered = append(filtered, r)
		}
	}
	em.replacements = filtered

	filtered2 := em.cycleReplacements[:0]
	for _, r := range em.cycleReplacements {
		if keep(r) {
			filtered2 = append(filtered2, r)
		}
	}
	em.cycleReplacements = filtered2
}

// ApplyReplacements runs the replacement pipeline on an action.
// Returns nil if the action was fully prevented/replaced.
// Each replacement fires at most once per event to prevent infinite loops.
func (em *EffectManager) ApplyReplacements(action Action, g *Game) Action {
	result, outcomes := em.prepareReplacements(action, g)
	applyPreventionLifeGains(g, outcomes)
	return result
}

func (em *EffectManager) prepareReplacements(action Action, g *Game, local ...ReplacementEffect) (Action, []preventionLifeGain) {
	switch action.(type) {
	case *DamageToPlayerAction, *DamageToCreatureAction:
		if damageActionAmount(action) <= 0 {
			return nil, nil
		}
		actions, gains := em.prepareDamageBatchReplacementsWithLocal([]Action{action}, g, local...)
		if len(actions) > 1 {
			return &expandedDamageAction{actions: actions, originalCount: 1}, gains
		}
		return actions[0], gains
	}
	return em.prepareReplacementsChoosing(action, g, chooseReplacement, local...)
}

func (em *EffectManager) prepareReplacementsChoosing(action Action, g *Game, choose func([]ReplacementEffect, Action, *Game) int, local ...ReplacementEffect) (Action, []preventionLifeGain) {
	var outcomes []preventionLifeGain
	applied := make(map[ReplacementEffect]bool)
	for action != nil {
		// Copy replacements can replace subtype state before the next entry check.
		if entry, ok := action.(*battlefieldEntryAction); ok && entry.context.entry.withoutSubtype != nil {
			entry.context.entry.withoutSubtype.apply(g, entry.permanent)
		}
		if entry, ok := action.(*battlefieldEntryAction); ok && g.isLandEntryProhibited(entry.permanent.HasType(TypeLand)) {
			return nil, outcomes
		}
		switch damage := action.(type) {
		case *DamageToPlayerAction:
			if damage.Amount() <= 0 {
				return nil, outcomes
			}
		case *DamageToCreatureAction:
			if damage.Amount() <= 0 {
				return nil, outcomes
			}
		}
		candidates := em.replacementCandidates(action, g, applied, local...)
		if len(candidates) == 0 {
			break
		}
		index := choose(candidates, action, g)
		r := candidates[index]
		applied[r] = true
		if source, ok := r.(*sourcePreventionReplacement); ok && source.amount != nil {
			var outcome preventionLifeGain
			action, outcome = source.preventAction(action)
			if outcome.amount > 0 {
				outcomes = append(outcomes, outcome)
			}
		} else {
			action = keepDamageCause(action, r.Replace(action, g))
		}
	}

	return action, outcomes
}

func (em *EffectManager) replacementCandidates(action Action, g *Game, applied map[ReplacementEffect]bool, local ...ReplacementEffect) []ReplacementEffect {
	var candidates []ReplacementEffect
	seen := make(map[ReplacementEffect]bool)
	for _, list := range [3][]ReplacementEffect{em.replacements, em.cycleReplacements, local} {
		for _, r := range list {
			if damageReplacementBlocked(action, g) {
				additional, ok := r.(interface{ appliesToUnpreventableDamage() bool })
				if !ok || !additional.appliesToUnpreventableDamage() {
					continue
				}
			}
			if !applied[r] && !seen[r] && r.IsActive(g) && r.Matches(action, g) {
				candidates = append(candidates, r)
				seen[r] = true
			}
		}
	}
	if entry, ok := action.(*battlefieldEntryAction); ok {
		for _, replacement := range append(append(entry.replacementCandidates(), entry.filteredReplacementCandidates(g)...), entry.retainedReplacementCandidates(g)...) {
			if !applied[replacement] {
				candidates = append(candidates, replacement)
			}
		}
	}
	if destruction, ok := action.(*DestroyPermanentAction); ok {
		for _, replacement := range destruction.staticRegenerationCandidates(g) {
			if !applied[replacement] {
				candidates = append(candidates, replacement)
			}
		}
	}
	// A replacement can change the recipient before protection applies.
	if damage, ok := action.(*DamageToCreatureAction); ok && !damageReplacementBlocked(action, g) {
		source := damage.DamageSource()
		if !source.Valid() {
			source = g.CaptureDamageSource(damage.ActionSource())
		}
		protection := &protectionDamageReplacement{source: source}
		if protection.Matches(action, g) {
			candidates = append(candidates, protection)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	bestPriority := ReplacementPriorityDefault
	for _, r := range candidates {
		p := getReplacementPriority(r)
		if p < bestPriority {
			bestPriority = p
		}
	}
	var tierCandidates []ReplacementEffect
	for _, r := range candidates {
		if getReplacementPriority(r) == bestPriority {
			tierCandidates = append(tierCandidates, r)
		}
	}
	candidates = tierCandidates

	return candidates
}

func chooseReplacement(candidates []ReplacementEffect, action Action, g *Game) int {
	index := 0
	if len(candidates) > 1 {
		if player := replacementAffectedPlayer(action, g); player != nil {
			labels := make([]string, len(candidates))
			for i, r := range candidates {
				labels[i] = replacementLabel(r, i, g)
			}
			index = player.ChooseMode(labels, "Choose a replacement or prevention effect")
			if index < 0 || index >= len(candidates) {
				index = 0
			}
		}
	}
	return index
}

func replacementLabel(r ReplacementEffect, index int, g *Game) string {
	label := fmt.Sprintf("Replacement effect %d", index+1)
	if card := g.FindCardAnywhere(r.SourceID()); card != nil {
		label += ": " + card.Name()
	}
	if description, ok := r.(interface{ Text() string }); ok {
		label += ": " + description.Text()
	}
	return label
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

func replacementAffectedPlayer(action Action, g *Game) Player {
	if a, ok := action.(interface{ PlayerID() uuid.UUID }); ok {
		return g.GetPlayer(a.PlayerID())
	}
	if a, ok := action.(interface{ PermanentID() uuid.UUID }); ok {
		if p := g.FindPermanent(a.PermanentID()); p != nil {
			return g.GetPlayer(p.ControllerID())
		}
		if c := g.FindCardAnywhere(a.PermanentID()); c != nil {
			return g.GetPlayer(c.Owner())
		}
	}
	return nil
}

func getReplacementPriority(r ReplacementEffect) ReplacementPriority {
	if p, ok := r.(ReplacementPriorityProvider); ok {
		prio := p.ReplacementPriority()
		if prio > 0 {
			return prio
		}
	}
	return ReplacementPriorityDefault
}

// DependencyDetector is an optional interface for continuous effects to declare dependencies under CR 613.8.
type DependencyDetector interface {
	DependsOn(other ContinuousEffect, g *Game) bool
}

func getEffectSublayer(e ContinuousEffect) SublayerPT {
	if sp, ok := e.(interface{ GetSublayer() SublayerPT }); ok {
		sub := sp.GetSublayer()
		if sub > 0 {
			return sub
		}
	}
	return SublayerPT_Modify
}

func orderEffectsByDependency(effects []ContinuousEffect, g *Game) []ContinuousEffect {
	order := effectDependencyOrder(effects, g)
	result := make([]ContinuousEffect, len(effects))
	for i, index := range order {
		result[i] = effects[index]
	}
	return result
}

func effectDependencyOrder(effects []ContinuousEffect, g *Game) []int {
	n := len(effects)
	original := make([]int, n)
	for i := range original {
		original[i] = i
	}
	if n <= 1 {
		return original
	}

	dependsOn := make([][]bool, n)
	for i := range dependsOn {
		dependsOn[i] = make([]bool, n)
	}

	hasAnyDep := false
	for i, effI := range effects {
		if detector, ok := effI.(DependencyDetector); ok {
			for j, effJ := range effects {
				if i != j && detector.DependsOn(effJ, g) {
					dependsOn[i][j] = true
					hasAnyDep = true
				}
			}
		}
	}

	if !hasAnyDep {
		return original
	}

	paths := make([][]bool, n)
	for i := range paths {
		paths[i] = slices.Clone(dependsOn[i])
	}
	for via := range n {
		for from := range n {
			for to := range n {
				paths[from][to] = paths[from][to] || paths[from][via] && paths[via][to]
			}
		}
	}
	for from := range n {
		for to := range n {
			if paths[from][to] && paths[to][from] {
				dependsOn[from][to] = false
			}
		}
	}

	inDegree := make([]int, n)
	for i := range n {
		for j := range n {
			if dependsOn[i][j] {
				inDegree[i]++
			}
		}
	}

	result := make([]int, 0, n)
	applied := make([]bool, n)

	for len(result) < n {
		next := -1
		for i := range n {
			if !applied[i] && inDegree[i] == 0 {
				next = i
				break
			}
		}
		if next == -1 {
			for i := range n {
				if !applied[i] {
					next = i
					break
				}
			}
		}
		applied[next] = true
		result = append(result, next)
		for i := range n {
			if dependsOn[i][next] {
				dependsOn[i][next] = false
				inDegree[i]--
			}
		}
	}

	return result
}

func (em *EffectManager) removeEndOfNextUpkeep(playerID uuid.UUID) {
	filtered := em.effects[:0]
	for _, effect := range em.effects {
		if captured, ok := effect.(*composedLayerEffect); ok && captured.endNextUpkeep && captured.upkeepReady && captured.upkeepPlayer == playerID {
			continue
		}
		filtered = append(filtered, effect)
	}
	em.effects = filtered
}

// Face-down base values stay fixed, but P/T effects must be read each pass.
func (p *Permanent) resetComputedPT() {
	p.cdaPower = nil
	p.cdaToughness = nil
	p.powerBonus = 0
	p.toughBonus = 0
	p.ptSwitched = false
}
