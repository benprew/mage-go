package mage

import "github.com/google/uuid"

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

// SimpleActivatedAbility is a basic activated ability.
type SimpleActivatedAbility struct {
	BaseAbility
	effects     []Effect
	costs       []Cost
	targets     []Target
	SorceryOnly bool
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
	for _, c := range a.costs {
		if !c.CanPay(a.source, controller, g) {
			return false
		}
	}
	return true
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
