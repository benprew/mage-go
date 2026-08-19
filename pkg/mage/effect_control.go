package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ControlConditionData determines whether a control effect remains active.
// Once any condition becomes false, the effect expires permanently.
type ControlConditionData interface {
	controlCondition(g *Game, source, target *Permanent, effectController uuid.UUID) bool
}

// ControlSourceExists requires the source to remain on the battlefield.
type ControlSourceExists struct{}

func (ControlSourceExists) controlCondition(_ *Game, source, _ *Permanent, _ uuid.UUID) bool {
	return source != nil
}

// ControlSourceControlledByEffectController requires the resolving ability's
// controller to retain control of the source.
type ControlSourceControlledByEffectController struct{}

func (ControlSourceControlledByEffectController) controlCondition(g *Game, source, _ *Permanent, controller uuid.UUID) bool {
	if source == nil {
		return false
	}
	sourceController := source.ControllerID()
	if reconciled, ok := g.layer2Controllers[source.ID()]; ok {
		sourceController = reconciled
	}
	return sourceController == controller
}

// ControlSourceTapped requires the source to remain tapped.
type ControlSourceTapped struct{}

func (ControlSourceTapped) controlCondition(_ *Game, source, _ *Permanent, _ uuid.UUID) bool {
	return source != nil && source.Tapped
}

// ControlTargetPowerLESource requires the target's power to be no greater
// than the source's power.
type ControlTargetPowerLESource struct{}

func (ControlTargetPowerLESource) controlCondition(g *Game, source, target *Permanent, _ uuid.UUID) bool {
	return source != nil && target != nil && target.CurrentPower(g) <= source.CurrentPower(g)
}

// ControlEffectSpec describes a Layer 2 control-changing effect discovered at
// resolution time.
type ControlEffectSpec struct {
	SourceID      uuid.UUID
	TargetID      uuid.UUID
	ControllerID  uuid.UUID
	Duration      Duration
	Conditions    []ControlConditionData
	TapMaintained bool
}

type controlContinuousEffect struct {
	targetID      uuid.UUID
	controllerID  uuid.UUID
	duration      Duration
	conditions    []ControlConditionData
	expired       bool
	tapMaintained bool
	effectSource
}

func (e *controlContinuousEffect) GetLayer() Layer       { return LayerControl }
func (e *controlContinuousEffect) GetDuration() Duration { return e.duration }

func (e *controlContinuousEffect) IsActive(g *Game) bool {
	return e.isActive(g, true)
}

func (e *controlContinuousEffect) isActive(g *Game, latchExpiration bool) bool {
	if e.expired {
		return false
	}
	target := g.FindPermanent(e.targetID)
	if target == nil {
		if latchExpiration {
			e.expired = true
		}
		return false
	}
	source := g.FindPermanent(e.SourceID())
	for _, condition := range e.conditions {
		if !condition.controlCondition(g, source, target, e.controllerID) {
			if latchExpiration {
				e.expired = true
			}
			return false
		}
	}
	return true
}

type attachedControlEffect struct{ effectSource }

func (e *attachedControlEffect) GetLayer() Layer       { return LayerControl }
func (e *attachedControlEffect) GetDuration() Duration { return WhileOnBattlefield }
func (e *attachedControlEffect) IsActive(g *Game) bool {
	source := g.FindPermanent(e.SourceID())
	return source != nil && source.IsAttached() && g.FindPermanent(source.AttachedTo) != nil
}
func (e *attachedControlEffect) Apply(g *Game) error {
	source := g.FindPermanent(e.SourceID())
	if source == nil {
		return nil
	}
	target := g.MutablePermanent(source.AttachedTo)
	if target == nil {
		return nil
	}
	controller := source.ControllerID()
	if reconciled, ok := g.layer2Controllers[source.ID()]; ok {
		controller = reconciled
	}
	target.computedController = controller
	return nil
}

func (e *controlContinuousEffect) Apply(g *Game) error {
	target := g.MutablePermanent(e.targetID)
	if target != nil {
		target.computedController = e.controllerID
	}
	return nil
}

func (e *controlContinuousEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.conditions = append([]ControlConditionData(nil), e.conditions...)
	return &clone
}

// AddControlEffect registers and immediately reconciles a Layer 2 control effect.
func (g *Game) AddControlEffect(spec ControlEffectSpec) {
	e := newControlContinuousEffect(spec)
	g.effects.Add(e)
	g.effects.Apply(g)
}

func newControlContinuousEffect(spec ControlEffectSpec) *controlContinuousEffect {
	e := &controlContinuousEffect{
		targetID:      spec.TargetID,
		controllerID:  spec.ControllerID,
		duration:      spec.Duration,
		conditions:    append([]ControlConditionData(nil), spec.Conditions...),
		tapMaintained: spec.TapMaintained,
	}
	e.SetSourceID(spec.SourceID)
	return e
}

// ExchangeControl atomically creates simultaneous indefinite Layer 2 effects
// that exchange the current controllers of two battlefield permanents.
func (g *Game) ExchangeControl(firstID, secondID, sourceID uuid.UUID) bool {
	first := g.FindPermanent(firstID)
	second := g.FindPermanent(secondID)
	if first == nil || second == nil || firstID == secondID {
		return false
	}
	firstController := first.ControllerID()
	secondController := second.ControllerID()
	if firstController == secondController || first.HasAttr(AttrCantChangeControl) || second.HasAttr(AttrCantChangeControl) {
		return false
	}
	g.effects.Add(newControlContinuousEffect(ControlEffectSpec{
		SourceID: sourceID, TargetID: firstID, ControllerID: secondController, Duration: Indefinite,
	}))
	g.effects.Add(newControlContinuousEffect(ControlEffectSpec{
		SourceID: sourceID, TargetID: secondID, ControllerID: firstController, Duration: Indefinite,
	}))
	g.effects.Apply(g)
	return true
}

type exchangeControlTargetsEffect struct {
	requireSharedPermanentType bool
}

// ExchangeControlOfTargets exchanges control of the first two targets if the
// entire exchange remains possible as the effect resolves.
func ExchangeControlOfTargets() Effect {
	return &exchangeControlTargetsEffect{}
}

// ExchangeControlOfTargetsSharingPermanentType additionally requires both
// targets to still share artifact, creature, or land type at resolution.
func ExchangeControlOfTargetsSharingPermanentType() Effect {
	return &exchangeControlTargetsEffect{requireSharedPermanentType: true}
}

func (*exchangeControlTargetsEffect) Text() string {
	return "exchange control of two target permanents"
}
func (*exchangeControlTargetsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeUnknown}
}

func (e *exchangeControlTargetsEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) < 2 || ctx.Targets[0] == uuid.Nil || ctx.Targets[1] == uuid.Nil {
		return nil
	}
	first := ctx.Game.FindPermanent(ctx.Targets[0])
	second := ctx.Game.FindPermanent(ctx.Targets[1])
	if first == nil || second == nil {
		return nil
	}
	if e.requireSharedPermanentType && !shareExchangePermanentType(first, second) {
		return nil
	}
	ctx.Game.ExchangeControl(first.ID(), second.ID(), ctx.SourceID)
	return nil
}

func shareExchangePermanentType(first, second *Permanent) bool {
	if first == nil || second == nil {
		return false
	}
	for _, cardType := range []CardType{TypeArtifact, TypeCreature, TypeLand} {
		if first.HasType(cardType) && second.HasType(cardType) {
			return true
		}
	}
	return false
}

// GainControlEffect creates continuous control effects when it resolves.
type GainControlEffect struct {
	selector      TargetSelector
	duration      Duration
	conditions    []ControlConditionData
	tapMaintained bool
}

// GainControl gains control of the target permanent indefinitely.
func GainControl() *GainControlEffect {
	return &GainControlEffect{selector: ToTarget(), duration: Indefinite}
}

func (e *GainControlEffect) Targeting(selector TargetSelector) *GainControlEffect {
	e.selector = selector
	return e
}

func (e *GainControlEffect) Until(duration Duration) *GainControlEffect {
	e.duration = duration
	return e
}

func (e *GainControlEffect) While(conditions ...ControlConditionData) *GainControlEffect {
	e.conditions = append(e.conditions, conditions...)
	return e
}

func (e *GainControlEffect) TapMaintained() *GainControlEffect {
	e.tapMaintained = true
	return e
}

func (e *GainControlEffect) Text() string { return "gain control of target permanent" }
func (e *GainControlEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *GainControlEffect) Apply(ctx *EffectContext) error {
	perms := resolvePermanents(ctx, e.selector)
	if len(perms) == 0 {
		return fmt.Errorf("no target for control change")
	}
	for _, perm := range perms {
		ctx.Game.AddControlEffect(ControlEffectSpec{
			SourceID:      ctx.SourceID,
			TargetID:      perm.ID(),
			ControllerID:  ctx.Controller,
			Duration:      e.duration,
			Conditions:    e.conditions,
			TapMaintained: e.tapMaintained,
		})
	}
	return nil
}

// ControlAttached continuously gives the Aura's controller control of the
// permanent it is attached to.
func ControlAttached() ContinuousEffect {
	return &attachedControlEffect{}
}

// ControlSourceByChosenPlayer continuously sets the source's controller to its
// chosen player.
func ControlSourceByChosenPlayer() ContinuousEffect {
	return FuncContinuousEffect(LayerControl, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		source := g.MutablePermanent(sourceID)
		if source != nil && source.ChosenPlayer != uuid.Nil {
			source.computedController = source.ChosenPlayer
		}
		return nil
	})
}
