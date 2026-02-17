package mage

import "github.com/google/uuid"

// ActivatedAbilityI is the interface for activated abilities.
type ActivatedAbilityI interface {
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
	Effs         []Effect
	Csts         []Cost
	Tgts         []Target
	SorceryOnly  bool
}

func NewActivatedAbility(effect Effect, cost Cost) *SimpleActivatedAbility {
	return &SimpleActivatedAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityActivated,
		},
		Effs: []Effect{effect},
		Csts: []Cost{cost},
	}
}

func (a *SimpleActivatedAbility) AddCost(c Cost) *SimpleActivatedAbility {
	a.Csts = append(a.Csts, c)
	return a
}

func (a *SimpleActivatedAbility) AddTarget(t Target) *SimpleActivatedAbility {
	a.Tgts = append(a.Tgts, t)
	return a
}

func (a *SimpleActivatedAbility) CanActivate(controller uuid.UUID, g *Game) bool {
	for _, c := range a.Csts {
		if !c.CanPay(a.Source_, controller, g) {
			return false
		}
	}
	return true
}

func (a *SimpleActivatedAbility) Effects() []Effect { return a.Effs }
func (a *SimpleActivatedAbility) Costs() []Cost     { return a.Csts }
func (a *SimpleActivatedAbility) Targets() []Target { return a.Tgts }
func (a *SimpleActivatedAbility) SorcerySpeed() bool { return a.SorceryOnly }

// EquipAbility is an activated ability for equipment (sorcery speed, targets creature you control).
type EquipAbility struct {
	SimpleActivatedAbility
}

func NewEquipAbility(cost Cost) *EquipAbility {
	ea := &EquipAbility{
		SimpleActivatedAbility: SimpleActivatedAbility{
			BaseAbility: BaseAbility{
				ID_:   uuid.New(),
				Type_: AbilityActivated,
			},
			Effs:        []Effect{AttachToTarget()},
			Csts:        []Cost{cost},
			Tgts:        []Target{TargetControlledCreature()},
			SorceryOnly: true,
		},
	}
	return ea
}
