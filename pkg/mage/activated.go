package mage

import (
	"github.com/google/uuid"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ActivatedAbility is the interface for activated abilities.
type ActivatedAbility interface {
	Ability
	CanActivate(controller uuid.UUID, g *Game) bool
	Effects() []Effect
	Costs() []Cost
	Targets() []Target
	SorcerySpeed() bool
}

// AbilityOption configures an activated ability during construction.
type AbilityOption func(*SimpleActivatedAbility)

// WithCost adds an additional cost to an activated ability.
func WithCost(c Cost) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.costs = append(a.costs, c)
	}
}

// WithTarget adds a target to an activated ability.
func WithTarget(t Target) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.targets = append(a.targets, t)
	}
}

// WithEffect adds an additional effect to an activated ability.
func WithEffect(e Effect) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.effects = append(a.effects, e)
	}
}

// WithSorcerySpeed marks an activated ability as "activate only as a sorcery"
// (CR 602.5d). ActivateAbilityByIndex enforces the timing restriction.
func WithSorcerySpeed() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.SorceryOnly = true
	}
}

// WithUpkeepOnly restricts an activated ability to only be activatable during an upkeep step.
func WithUpkeepOnly() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.UpkeepOnly = true
	}
}

// WithStepOnly restricts an activated ability to only be activatable during the given step.
func WithStepOnly(step PhaseStep) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.StepOnly = step
	}
}

// WithYourTurnOnly restricts an activated ability to only be activatable during its controller's turn.
func WithYourTurnOnly() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.YourTurnOnly = true
	}
}

// WithOncePerTurn restricts an activated ability to once per turn.
func WithOncePerTurn() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.OncePerTurn = true
	}
}

// WithMaxActivationsPerTurn restricts an activated ability to n activations per turn.
func WithMaxActivationsPerTurn(n int) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.MaxActivationsPerTurn = n
	}
}

// WithAnyPlayerMay allows any player (not just the controller) to activate the ability.
func WithAnyPlayerMay() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.AnyPlayerMayUse = true
	}
}

// WithOpponentOnlyMay allows only the opponents of the controller to activate the ability.
// The controller cannot activate it.
func WithOpponentOnlyMay() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.AnyPlayerMayUse = true
		a.OpponentOnlyMayUse = true
	}
}

// WithControlledSinceTurnStart restricts activation to only when the source has been
// continuously controlled since the beginning of the controller's most recent turn.
func WithControlledSinceTurnStart() AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.ControlledSinceTurnStart = true
	}
}

// ActivationCondition is a predicate that returns true if the activation
// should be permitted. It runs in addition to the standard timing/cost
// gates and is consulted by CanActivate.
type ActivationCondition func(g *Game, src *Permanent, controller uuid.UUID) bool

// WithActivationCondition adds a card-defined predicate gating activation
// (e.g. "activate only if this creature's power is 4 or greater"). The
// predicate is invoked with the (live) source permanent — pass nil-safe
// logic, since the source may be missing if it left the battlefield.
func WithActivationCondition(cond ActivationCondition) AbilityOption {
	return func(a *SimpleActivatedAbility) {
		a.activationConds = append(a.activationConds, cond)
	}
}

// SimpleActivatedAbility is a basic activated ability.
type SimpleActivatedAbility struct {
	BaseAbility
	effects                  []Effect
	costs                    []Cost
	targets                  []Target
	SorceryOnly              bool
	UpkeepOnly               bool      // Can only be activated during an upkeep step
	YourTurnOnly             bool      // Can only be activated during controller's turn
	StepOnly                 PhaseStep // If non-zero, can only be activated during this step
	OncePerTurn              bool      // Can only be activated once per turn
	MaxActivationsPerTurn    int       // Max activations per turn (0 = unlimited, overrides OncePerTurn)
	AnyPlayerMayUse          bool      // Any player may activate this ability
	OpponentOnlyMayUse       bool      // Only opponents of the controller may activate
	ControlledSinceTurnStart bool      // Only if controlled since beginning of most recent turn
	activatedThisTurn        bool      // Tracks whether this ability has been activated this turn
	activationsThisTurn      int       // Counts activations for MaxActivationsPerTurn
	activationConds          []ActivationCondition
}

// NewActivatedAbility creates an activated ability with a primary effect, a primary cost,
// and optional additional costs, targets, or effects via AbilityOption functions.
func NewActivatedAbility(effect Effect, cost Cost, opts ...AbilityOption) *SimpleActivatedAbility {
	a := &SimpleActivatedAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilityActivated,
		},
		effects: []Effect{effect},
		costs:   []Cost{cost},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *SimpleActivatedAbility) CanActivate(controller uuid.UUID, g *Game) bool {
	if perm := g.FindPermanent(a.source); perm != nil && perm.HasAttr(AttrCantActivateNonManaAbilities) {
		return false
	}
	if a.UpkeepOnly && g.step != Upkeep {
		return false
	}
	if a.YourTurnOnly && g.ActivePlayerObj().PlayerID() != controller {
		return false
	}
	if a.StepOnly != 0 && g.step != a.StepOnly {
		return false
	}
	if a.MaxActivationsPerTurn > 0 && a.activationsThisTurn >= a.MaxActivationsPerTurn {
		return false
	}
	if a.OncePerTurn && a.activatedThisTurn {
		return false
	}
	if a.ControlledSinceTurnStart {
		perm := g.FindPermanent(a.source)
		if perm == nil || perm.TurnControlGained >= g.turn {
			return false
		}
	}
	if len(a.activationConds) > 0 {
		src := g.FindPermanent(a.source)
		for _, cond := range a.activationConds {
			if !cond(g, src, controller) {
				return false
			}
		}
	}
	for _, c := range a.costs {
		if !c.CanPay(a.source, controller, g) {
			return false
		}
	}
	return true
}

// MarkActivated sets the once-per-turn flag. Called after the ability is paid for.
func (a *SimpleActivatedAbility) MarkActivated() {
	if a.OncePerTurn {
		a.activatedThisTurn = true
	}
	if a.MaxActivationsPerTurn > 0 {
		a.activationsThisTurn++
	}
}

// ResetActivation clears the once-per-turn flag. Called at the beginning of each turn.
func (a *SimpleActivatedAbility) ResetActivation() {
	a.activatedThisTurn = false
	a.activationsThisTurn = 0
}

// IsAnyPlayerAbility returns true if any player may activate this ability.
func (a *SimpleActivatedAbility) IsAnyPlayerAbility() bool {
	return a.AnyPlayerMayUse
}

func (a *SimpleActivatedAbility) Effects() []Effect  { return a.effects }
func (a *SimpleActivatedAbility) Costs() []Cost      { return a.costs }
func (a *SimpleActivatedAbility) Targets() []Target  { return a.targets }
func (a *SimpleActivatedAbility) SorcerySpeed() bool { return a.SorceryOnly }

// EquipAbility is an activated ability for equipment (sorcery speed, targets creature you control).
type EquipAbility struct {
	SimpleActivatedAbility
}

// NewEquipAbility creates a sorcery-speed activated ability that attaches the source
// equipment to a target creature the controller owns. Used by Equipment cards.
func NewEquipAbility(cost Cost) *EquipAbility {
	ea := &EquipAbility{
		SimpleActivatedAbility: SimpleActivatedAbility{
			BaseAbility: BaseAbility{
				id:          uuid.New(),
				abilityType: AbilityActivated,
			},
			effects:     []Effect{AttachToTarget()},
			costs:       []Cost{cost},
			targets:     []Target{TargetControlledCreature()},
			SorceryOnly: true,
		},
	}
	return ea
}
