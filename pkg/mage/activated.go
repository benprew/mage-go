package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// ActionKind identifies whether an action is cast as a spell or activated from
// a permanent. Mana abilities remain separate because they do not use the stack.
type ActionKind int

const (
	ActionSpell ActionKind = iota
	ActionActivated
)

// TimingRule describes when an action can be used.
type TimingRule int

const (
	TimingInstant TimingRule = iota
	TimingSorcery
	TimingUpkeepOnly
	TimingStepOnly
	YourTurnOnly
)

// ActivationLimits stores reusable activation limits for activated actions.
type ActivationLimits struct {
	OncePerTurn              bool
	MaxActivationsPerTurn    int
	ControlledSinceTurnStart bool
}

// ActivationPermission stores who may activate an activated action.
type ActivationPermission struct {
	AnyPlayerMayUse    bool
	OpponentOnlyMayUse bool
}

// ActivatedAbility is the interface for activated abilities.
type ActivatedAbility interface {
	Ability
	CanActivate(controller uuid.UUID, g *Game) bool
	Effects() []Effect
	Costs() []Cost
	Targets() []Target
	SorcerySpeed() bool
}

// ActionOption configures a spell or activated ability action.
type ActionOption func(*ActionDefinition)

// AbilityOption is kept for compatibility with existing activated ability helpers.
type AbilityOption = ActionOption

// WithCost adds a cost to an action.
func WithCost(c Cost) ActionOption {
	return func(a *ActionDefinition) {
		a.costs = append(a.costs, c)
	}
}

// WithTarget adds a target to an action.
func WithTarget(t Target) ActionOption {
	return func(a *ActionDefinition) {
		a.targets = append(a.targets, t)
	}
}

// WithEffect adds an effect to an action.
func WithEffect(e Effect) ActionOption {
	return func(a *ActionDefinition) {
		a.effects = append(a.effects, e)
	}
}

// WithEffects adds multiple effects to an action.
func WithEffects(effects ...Effect) ActionOption {
	return func(a *ActionDefinition) {
		a.effects = append(a.effects, effects...)
	}
}

// WithTiming sets the timing rule for an action.
func WithTiming(rule TimingRule) ActionOption {
	return func(a *ActionDefinition) {
		a.timing = rule
		a.UpkeepOnly = rule == TimingUpkeepOnly
	}
}

// WithStepTiming restricts an action to a specific step.
func WithStepTiming(step PhaseStep) ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingStepOnly
		a.stepOnly = step
		a.StepOnly = step
	}
}

// WithSorcerySpeed marks an activated ability as "activate only as a sorcery"
// (CR 602.5d). ActivateAbilityByIndex enforces the timing restriction.
func WithSorcerySpeed() ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingSorcery
	}
}

// WithUpkeepOnly restricts an activated ability to only be activatable during an upkeep step.
func WithUpkeepOnly() ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingUpkeepOnly
		a.UpkeepOnly = true
	}
}

// WithStepOnly restricts an activated ability to only be activatable during the given step.
func WithStepOnly(step PhaseStep) ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingStepOnly
		a.stepOnly = step
		a.StepOnly = step
	}
}

// WithYourTurnOnly restricts an activated ability to only be activatable during its controller's turn.
func WithYourTurnOnly() ActionOption {
	return func(a *ActionDefinition) {
		a.timing = YourTurnOnly
	}
}

// WithOncePerTurn restricts an activated ability to once per turn.
func WithOncePerTurn() ActionOption {
	return func(a *ActionDefinition) {
		a.limits.OncePerTurn = true
		a.OncePerTurn = true
	}
}

// WithMaxActivationsPerTurn restricts an activated ability to n activations per turn.
func WithMaxActivationsPerTurn(n int) ActionOption {
	return func(a *ActionDefinition) {
		a.limits.MaxActivationsPerTurn = n
		a.MaxActivationsPerTurn = n
	}
}

// WithAnyPlayerMay allows any player (not just the controller) to activate the ability.
func WithAnyPlayerMay() ActionOption {
	return func(a *ActionDefinition) {
		a.permission.AnyPlayerMayUse = true
		a.AnyPlayerMayUse = true
	}
}

// WithOpponentOnlyMay allows only the opponents of the controller to activate the ability.
// The controller cannot activate it.
func WithOpponentOnlyMay() ActionOption {
	return func(a *ActionDefinition) {
		a.permission.AnyPlayerMayUse = true
		a.permission.OpponentOnlyMayUse = true
		a.AnyPlayerMayUse = true
		a.OpponentOnlyMayUse = true
	}
}

// WithControlledSinceTurnStart restricts activation to only when the source has been
// continuously controlled since the beginning of the controller's most recent turn.
func WithControlledSinceTurnStart() ActionOption {
	return func(a *ActionDefinition) {
		a.limits.ControlledSinceTurnStart = true
		a.ControlledSinceTurnStart = true
	}
}

// WithActivationLimit applies activation limits to an action.
func WithActivationLimit(limits ActivationLimits) ActionOption {
	return func(a *ActionDefinition) {
		a.limits = limits
		a.OncePerTurn = limits.OncePerTurn
		a.MaxActivationsPerTurn = limits.MaxActivationsPerTurn
		a.ControlledSinceTurnStart = limits.ControlledSinceTurnStart
	}
}

// WithActivationPermission applies activation permissions to an action.
func WithActivationPermission(permission ActivationPermission) ActionOption {
	return func(a *ActionDefinition) {
		a.permission = permission
		a.AnyPlayerMayUse = permission.AnyPlayerMayUse
		a.OpponentOnlyMayUse = permission.OpponentOnlyMayUse
	}
}

// ActionDefinition is the shared resolvable action model for spells and
// non-mana activated abilities.
type ActionDefinition struct {
	BaseAbility
	kind                     ActionKind
	effects                  []Effect
	costs                    []Cost
	targets                  []Target
	timing                   TimingRule
	stepOnly                 PhaseStep
	limits                   ActivationLimits
	permission               ActivationPermission
	UpkeepOnly               bool      // legacy mirror for compatibility
	StepOnly                 PhaseStep // legacy mirror for compatibility
	OncePerTurn              bool      // legacy mirror for compatibility
	MaxActivationsPerTurn    int       // legacy mirror for compatibility
	AnyPlayerMayUse          bool      // legacy mirror for compatibility
	OpponentOnlyMayUse       bool      // legacy mirror for compatibility
	ControlledSinceTurnStart bool      // legacy mirror for compatibility
	activatedThisTurn        bool      // Tracks whether this ability has been activated this turn
	activationsThisTurn      int       // Counts activations for MaxActivationsPerTurn
}

// SimpleActivatedAbility is kept as a compatibility name for activated actions.
type SimpleActivatedAbility = ActionDefinition

// NewAction creates an action definition from options.
func NewAction(kind ActionKind, opts ...ActionOption) *ActionDefinition {
	abilityType := AbilitySpell
	if kind == ActionActivated {
		abilityType = AbilityActivated
	}
	a := &ActionDefinition{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: abilityType,
		},
		kind:   kind,
		timing: TimingInstant,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// NewActivated creates an activated action with a primary cost and effects.
func NewActivated(cost Cost, parts ...any) *ActionDefinition {
	opts := []ActionOption{WithCost(cost)}
	opts = append(opts, actionPartsToOptions(parts...)...)
	return NewAction(ActionActivated, opts...)
}

// NewActivatedAbility creates an activated ability with a primary effect, a primary cost,
// and optional additional costs, targets, or effects via AbilityOption functions.
func NewActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) *SimpleActivatedAbility {
	parts := []any{effect}
	for _, opt := range opts {
		parts = append(parts, opt)
	}
	return NewActivated(cost, parts...)
}

func actionPartsToOptions(parts ...any) []ActionOption {
	opts := make([]ActionOption, 0, len(parts))
	for _, part := range parts {
		switch v := part.(type) {
		case nil:
		case Effect:
			opts = append(opts, WithEffect(v))
		case []Effect:
			opts = append(opts, WithEffects(v...))
		case ActionOption:
			opts = append(opts, v)
		default:
			panic("mage: unsupported action builder argument")
		}
	}
	return opts
}

// Kind returns whether this action is a spell or activated ability.
func (a *ActionDefinition) Kind() ActionKind { return a.kind }

func (a *ActionDefinition) CanActivate(controller uuid.UUID, g *Game) bool {
	if (a.timing == TimingUpkeepOnly || a.UpkeepOnly) && g.step != Upkeep {
		return false
	}
	if a.timing == YourTurnOnly && g.ActivePlayerObj().PlayerID() != controller {
		return false
	}
	stepOnly := a.StepOnly
	if a.timing == TimingStepOnly {
		stepOnly = a.stepOnly
	}
	if stepOnly != 0 && g.step != stepOnly {
		return false
	}
	maxActivations := a.MaxActivationsPerTurn
	if a.limits.MaxActivationsPerTurn > 0 {
		maxActivations = a.limits.MaxActivationsPerTurn
	}
	if maxActivations > 0 && a.activationsThisTurn >= maxActivations {
		return false
	}
	if (a.OncePerTurn || a.limits.OncePerTurn) && a.activatedThisTurn {
		return false
	}
	if a.ControlledSinceTurnStart || a.limits.ControlledSinceTurnStart {
		perm := g.FindPermanent(a.source)
		if perm == nil || perm.TurnControlGained >= g.turn {
			return false
		}
	}
	for _, c := range a.costs {
		if !c.CanPay(a.source, controller, g) {
			return false
		}
	}

	// CR 307.5 / 602.5d — "activate only as a sorcery" means the ability can
	// only be activated when its controller could cast a sorcery: main phase,
	// active player, empty stack.
	if a.SorcerySpeed() {
		if !g.step.IsMainPhase() {
			return false
		}
		if g.ActivePlayerObj().PlayerID() != controller {
			return false
		}
		if !g.stack.IsEmpty() {
			return false
		}
	}

	return true
}

// MarkActivated sets the once-per-turn flag. Called after the ability is paid for.
func (a *ActionDefinition) MarkActivated() {
	if a.OncePerTurn || a.limits.OncePerTurn {
		a.activatedThisTurn = true
	}
	if a.MaxActivationsPerTurn > 0 || a.limits.MaxActivationsPerTurn > 0 {
		a.activationsThisTurn++
	}
}

// ResetActivation clears the once-per-turn flag. Called at the beginning of each turn.
func (a *ActionDefinition) ResetActivation() {
	a.activatedThisTurn = false
	a.activationsThisTurn = 0
}

// IsAnyPlayerAbility returns true if any player may activate this ability.
func (a *ActionDefinition) IsAnyPlayerAbility() bool {
	return a.AnyPlayerMayUse || a.permission.AnyPlayerMayUse
}

func (a *ActionDefinition) Effects() []Effect  { return a.effects }
func (a *ActionDefinition) Costs() []Cost      { return a.costs }
func (a *ActionDefinition) Targets() []Target  { return a.targets }
func (a *ActionDefinition) SorcerySpeed() bool { return a.timing == TimingSorcery }

// EquipAbility is an activated ability for equipment (sorcery speed, targets creature you control).
type EquipAbility struct {
	ActionDefinition
}

// NewEquipAbility creates a sorcery-speed activated ability that attaches the source
// equipment to a target creature the controller owns. Used by Equipment cards.
func NewEquipAbility(cost Cost) *EquipAbility {
	action := NewActivated(
		cost,
		AttachToTarget(),
		WithTarget(TargetControlledCreature()),
		WithSorcerySpeed(),
	)
	return &EquipAbility{
		ActionDefinition: *action,
	}
}
