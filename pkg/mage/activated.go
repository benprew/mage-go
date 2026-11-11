package mage

import (
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ActionKind identifies spells and activated actions.
// Mana abilities resolve without the stack.
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
	CanActivate(ctx AbilityContext, g *Game) bool
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

// WithAIHint attaches explicit strategy metadata to a spell or activated ability.
func WithAIHint(hint AIHint) ActionOption {
	return func(a *ActionDefinition) {
		a.aiHints = append(a.aiHints, hint)
	}
}

// WithTiming sets the timing rule for an action.
func WithTiming(rule TimingRule) ActionOption {
	return func(a *ActionDefinition) { a.timing = rule }
}

// WithStepTiming restricts an action to a specific step.
func WithStepTiming(step PhaseStep) ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingStepOnly
		a.stepOnly = step
	}
}

// WithSorcerySpeed marks an activated ability as "activate only as a sorcery"
// (CR 602.5d). ActivateAbilityByIndex enforces the timing restriction.
func WithSorcerySpeed() ActionOption {
	return func(a *ActionDefinition) { a.timing = TimingSorcery }
}

// WithUpkeepOnly restricts an activated ability to only be activatable during an upkeep step.
func WithUpkeepOnly() ActionOption {
	return func(a *ActionDefinition) { a.timing = TimingUpkeepOnly }
}

// WithStepOnly restricts an activated ability to only be activatable during the given step.
func WithStepOnly(step PhaseStep) ActionOption {
	return func(a *ActionDefinition) {
		a.timing = TimingStepOnly
		a.stepOnly = step
	}
}

// WithYourTurnOnly restricts an activated ability to only be activatable during its controller's turn.
func WithYourTurnOnly() ActionOption {
	return func(a *ActionDefinition) { a.timing = YourTurnOnly }
}

// WithOncePerTurn restricts an activated ability to once per turn.
func WithOncePerTurn() ActionOption {
	return func(a *ActionDefinition) { a.limits.OncePerTurn = true }
}

// WithMaxActivationsPerTurn restricts an activated ability to n activations per turn.
func WithMaxActivationsPerTurn(n int) ActionOption {
	return func(a *ActionDefinition) { a.limits.MaxActivationsPerTurn = n }
}

// WithAnyPlayerMay allows any player (not just the controller) to activate the ability.
func WithAnyPlayerMay() ActionOption {
	return func(a *ActionDefinition) { a.permission.AnyPlayerMayUse = true }
}

// WithOpponentOnlyMay allows only the opponents of the controller to activate the ability.
// The controller cannot activate it.
func WithOpponentOnlyMay() ActionOption {
	return func(a *ActionDefinition) {
		a.permission.AnyPlayerMayUse = true
		a.permission.OpponentOnlyMayUse = true
	}
}

// WithControlledSinceTurnStart restricts activation to only when the source has been
// continuously controlled since the beginning of the controller's most recent turn.
func WithControlledSinceTurnStart() ActionOption {
	return func(a *ActionDefinition) { a.limits.ControlledSinceTurnStart = true }
}

// ActivationCondition is a predicate evaluated when checking whether an
// activated ability may be activated. It returns true if activation is
// allowed. The source permanent may be nil if the source is not on the
// battlefield.
type ActivationCondition func(g *Game, source *Permanent, controller uuid.UUID) bool

// WithActivationCondition adds a card-defined predicate that must return true
// for the ability to be activatable. Multiple conditions are AND-ed.
func WithActivationCondition(cond ActivationCondition) ActionOption {
	return func(a *ActionDefinition) {
		a.activationConds = append(a.activationConds, cond)
	}
}

// WithActivationLimit applies activation limits to an action.
func WithActivationLimit(limits ActivationLimits) ActionOption {
	return func(a *ActionDefinition) { a.limits = limits }
}

// WithActivationPermission applies activation permissions to an action.
func WithActivationPermission(permission ActivationPermission) ActionOption {
	return func(a *ActionDefinition) { a.permission = permission }
}

// WithActivationZone restricts the zone from which an activated ability may be activated.
func WithActivationZone(zone Zone) ActionOption {
	return func(a *ActionDefinition) { a.zone = zone }
}

// ActionDefinition stores the specification for a spell or activated ability.
type ActionDefinition struct {
	BaseAbility
	kind            ActionKind
	zone            Zone
	effects         []Effect
	costs           []Cost
	targets         []Target
	timing          TimingRule
	stepOnly        PhaseStep
	limits          ActivationLimits
	permission      ActivationPermission
	activationConds []ActivationCondition
	aiHints         []AIHint
	modes           []Mode
	legacyModes     bool
	loyalty         bool
}

// ActivationZone returns the zone from which this ability can be activated.
func (a *ActionDefinition) ActivationZone() Zone {
	if a.zone == 0 {
		return ZoneBattlefield
	}
	return a.zone
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
		id:          uuid.New(),
		abilityType: abilityType,
		kind:        kind,
		timing:      TimingInstant,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// NewActivated creates an activated action with a primary cost and effects.
func NewActivated(cost Cost, parts ...any) *ActionDefinition {
	opts := make([]ActionOption, 1, 1+len(parts))
	opts[0] = WithCost(cost)
	opts = append(opts, actionPartsToOptions(parts...)...)
	return NewAction(ActionActivated, opts...)
}

// NewModalActivated creates an activated ability whose chosen mode determines
// its targets and effects. Additional parts may add costs or action options.
func NewModalActivated(cost Cost, modes []Mode, parts ...any) *ActionDefinition {
	if len(modes) < 2 {
		panic("NewModalActivated: a modal ability needs at least two modes (CR 700.2)")
	}
	action := NewActivated(cost, parts...)
	action.modes = cloneModes(modes)
	return action
}

// NewActivatedAbility creates an activated ability with a primary effect, a primary cost,
// and optional additional costs, targets, or effects via AbilityOption functions.
func NewActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) *SimpleActivatedAbility {
	parts := make([]any, 1, 1+len(opts))
	parts[0] = effect
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

func (a *ActionDefinition) CanActivate(ctx AbilityContext, g *Game) bool {
	if !a.canActivateWithoutPayingCosts(ctx, g, a.Type() == AbilityMana) {
		return false
	}
	for _, c := range a.costs {
		if !c.CanPay(ctx.SourceID, ctx.Controller, g) {
			return false
		}
	}
	return true
}

func (a *ActionDefinition) canActivateWithoutPayingCosts(ctx AbilityContext, g *Game, isManaAbility bool) bool {
	controller := ctx.Controller
	if a.timing == TimingUpkeepOnly && g.turns.Step() != Upkeep {
		return false
	}
	if perm := g.FindPermanent(ctx.SourceID); perm != nil {
		if a.ActivationZone() != ZoneBattlefield {
			return false
		}
		if perm.HasAttr(AttrCantActivate) || (!isManaAbility && perm.HasAttr(AttrCantActivateNonManaAbilities)) {
			return false
		}
		if perm.ControllerID() != controller && !a.permission.AnyPlayerMayUse {
			return false
		}
		if perm.ControllerID() == controller && a.permission.OpponentOnlyMayUse {
			return false
		}
	} else if isManaAbility {
		return false
	}
	if a.timing == YourTurnOnly && g.ActivePlayerObj().PlayerID() != controller {
		return false
	}
	if a.timing == TimingStepOnly && a.stepOnly != 0 && g.turns.Step() != a.stepOnly {
		return false
	}
	if a.limits.MaxActivationsPerTurn > 0 && g.activationCount(ctx.SourceID, a.AbilityID(), ctx.AbilityIndex) >= a.limits.MaxActivationsPerTurn {
		return false
	}
	if a.limits.OncePerTurn && (g.activationCount(ctx.SourceID, a.AbilityID(), ctx.AbilityIndex) > 0) {
		return false
	}
	if a.limits.ControlledSinceTurnStart {
		perm := g.FindPermanent(ctx.SourceID)
		if perm == nil || !perm.ControlledSinceTurnStart(g) {
			return false
		}
	}
	if len(a.activationConds) > 0 {
		src := g.FindPermanent(ctx.SourceID)
		for _, cond := range a.activationConds {
			if !cond(g, src, controller) {
				return false
			}
		}
	}

	// CR 307.5 / 602.5d — "activate only as a sorcery" means the ability can
	// only be activated when its controller could cast a sorcery: main phase,
	// active player, empty stack.
	if a.SorcerySpeed() {
		if !g.turns.Step().IsMainPhase() {
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

// IsAnyPlayerAbility returns true if any player may activate this ability.
func (a *ActionDefinition) IsAnyPlayerAbility() bool {
	return a.permission.AnyPlayerMayUse
}

// IsOpponentOnlyAbility returns true if only opponents (not the controller)
// may activate this ability.
func (a *ActionDefinition) IsOpponentOnlyAbility() bool {
	return a.permission.OpponentOnlyMayUse
}

func (a *ActionDefinition) Effects() []Effect  { return slices.Clone(a.effects) }
func (a *ActionDefinition) Costs() []Cost      { return slices.Clone(a.costs) }
func (a *ActionDefinition) Targets() []Target  { return slices.Clone(a.targets) }
func (a *ActionDefinition) SorcerySpeed() bool { return a.timing == TimingSorcery }
func (a *ActionDefinition) AIHints() []AIHint {
	hints := slices.Clone(a.aiHints)
	for i := range hints {
		hints[i].Roles = slices.Clone(hints[i].Roles)
	}
	return hints
}
func (a *ActionDefinition) Modes() []Mode { return cloneModes(a.modes) }

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

// WithLoyaltyAbility prevents mana classification for a loyalty ability.
func WithLoyaltyAbility() ActionOption { return func(a *ActionDefinition) { a.loyalty = true } }

// Type applies the mana ability criteria in CR 605.1a.
func (a *ActionDefinition) Type() AbilityType {
	if a.kind != ActionActivated {
		return AbilitySpell
	}
	if a.loyalty || len(a.targets) > 0 {
		return AbilityActivated
	}
	for _, mode := range a.modes {
		if len(mode.Targets) > 0 {
			return AbilityActivated
		}
	}
	for _, effect := range a.effects {
		if _, ok := effect.(ManaProductionEffect); ok {
			return AbilityMana
		}
	}
	for _, mode := range a.modes {
		for _, effect := range mode.Effects {
			if _, ok := effect.(ManaProductionEffect); ok {
				return AbilityMana
			}
		}
	}
	return AbilityActivated
}
