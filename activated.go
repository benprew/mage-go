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

// SimpleActivatedAbility is a basic activated ability.
type SimpleActivatedAbility struct {
	BaseAbility
	effects         []Effect
	costs         []Cost
	targets         []Target
	SorceryOnly  bool
}

func NewActivatedAbility(effect Effect, cost Cost) *SimpleActivatedAbility {
	return &SimpleActivatedAbility{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityActivated,
		},
		effects: []Effect{effect},
		costs: []Cost{cost},
	}
}

func (a *SimpleActivatedAbility) AddCost(c Cost) *SimpleActivatedAbility {
	a.costs = append(a.costs, c)
	return a
}

func (a *SimpleActivatedAbility) AddEffect(e Effect) *SimpleActivatedAbility {
	a.effects = append(a.effects, e)
	return a
}

func (a *SimpleActivatedAbility) AddTarget(t Target) *SimpleActivatedAbility {
	a.targets = append(a.targets, t)
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

func (a *SimpleActivatedAbility) Effects() []Effect { return a.effects }
func (a *SimpleActivatedAbility) Costs() []Cost     { return a.costs }
func (a *SimpleActivatedAbility) Targets() []Target { return a.targets }
func (a *SimpleActivatedAbility) SorcerySpeed() bool { return a.SorceryOnly }

// EquipAbility is an activated ability for equipment (sorcery speed, targets creature you control).
type EquipAbility struct {
	SimpleActivatedAbility
}

func NewEquipAbility(cost Cost) *EquipAbility {
	ea := &EquipAbility{
		SimpleActivatedAbility: SimpleActivatedAbility{
			BaseAbility: BaseAbility{
				id:   uuid.New(),
				abilityType: AbilityActivated,
			},
			effects:        []Effect{AttachToTarget()},
			costs:        []Cost{cost},
			targets:        []Target{TargetControlledCreature()},
			SorceryOnly: true,
		},
	}
	return ea
}
