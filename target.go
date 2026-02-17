package mage

import "github.com/google/uuid"

// Target represents a targeting requirement.
type Target interface {
	Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID
	Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error
	Chosen() []uuid.UUID
	IsChosen() bool
	Min() int
	Max() int
	Reset()
}

// BaseTarget provides common target fields.
type BaseTarget struct {
	chosen []uuid.UUID
	min    int
	max    int
}

func (t *BaseTarget) Chosen() []uuid.UUID { return t.chosen }
func (t *BaseTarget) IsChosen() bool      { return len(t.chosen) > 0 }
func (t *BaseTarget) Min() int            { return t.min }
func (t *BaseTarget) Max() int            { return t.max }
func (t *BaseTarget) Reset()              { t.chosen = nil }

// CreatureTarget targets a creature on the battlefield.
type CreatureTarget struct {
	BaseTarget
	Filters []PermanentFilter
}

func TargetCreature(filters ...PermanentFilter) Target {
	return &CreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

func (t *CreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if !p.HasType(TypeCreature) {
			continue
		}
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f(p, g) {
				match = false
				break
			}
		}
		if match {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *CreatureTarget) Choose(controller uuid.UUID, sourceCard Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// PlayerTarget targets a player.
type PlayerTarget struct {
	BaseTarget
}

func TargetPlayer() Target {
	return &PlayerTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *PlayerTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *PlayerTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// AnyTarget targets a creature or player.
type AnyTargetImpl struct {
	BaseTarget
}

func TargetAnyTarget() Target {
	return &AnyTargetImpl{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *AnyTargetImpl) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeCreature) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	for _, p := range g.Players {
		result = append(result, p.PlayerID())
	}
	return result
}

func (t *AnyTargetImpl) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// GraveyardCreatureTarget targets a creature card in your graveyard.
type GraveyardCreatureTarget struct {
	BaseTarget
}

func TargetCreatureInYourGraveyard() Target {
	return &GraveyardCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *GraveyardCreatureTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Graveyard() {
		if IsCreatureCard(c) {
			result = append(result, c.ID())
		}
	}
	return result
}

func (t *GraveyardCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ControlledCreatureTarget targets a creature you control.
type ControlledCreatureTarget struct {
	BaseTarget
}

func TargetControlledCreature() Target {
	return &ControlledCreatureTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ControlledCreatureTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeCreature) && p.Controller == controller && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ControlledCreatureTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// PermanentTarget targets any permanent on the battlefield with optional filters.
type PermanentTarget struct {
	BaseTarget
	Filters []PermanentFilter
}

func TargetPermanent(filters ...PermanentFilter) Target {
	return &PermanentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
		Filters:    filters,
	}
}

func (t *PermanentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if !p.CanBeTargetedBy(sourceCard, controller, g) {
			continue
		}
		match := true
		for _, f := range t.Filters {
			if !f(p, g) {
				match = false
				break
			}
		}
		if match {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *PermanentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// LandTarget targets a land on the battlefield.
type LandTarget struct {
	BaseTarget
}

func TargetLand() Target {
	return &LandTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *LandTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeLand) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *LandTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ArtifactTarget targets an artifact on the battlefield.
type ArtifactTarget struct {
	BaseTarget
}

func TargetArtifact() Target {
	return &ArtifactTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ArtifactTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if p.HasType(TypeArtifact) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ArtifactTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// ArtifactOrEnchantmentTarget targets an artifact or enchantment.
type ArtifactOrEnchantmentTarget struct {
	BaseTarget
}

func TargetArtifactOrEnchantment() Target {
	return &ArtifactOrEnchantmentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *ArtifactOrEnchantmentTarget) Possible(controller uuid.UUID, sourceCard Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Battlefield {
		if (p.HasType(TypeArtifact) || p.HasType(TypeEnchantment)) && p.CanBeTargetedBy(sourceCard, controller, g) {
			result = append(result, p.ID())
		}
	}
	return result
}

func (t *ArtifactOrEnchantmentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// SpellOnStackTarget targets a spell on the stack.
type SpellOnStackTarget struct {
	BaseTarget
}

func TargetSpellOnStack() Target {
	return &SpellOnStackTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *SpellOnStackTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.Stack.Objects() {
		if !obj.IsAbility && obj.Card != nil {
			result = append(result, obj.SourceID)
		}
	}
	return result
}

func (t *SpellOnStackTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// GraveyardCardTarget targets any card in your graveyard (not just creatures).
type GraveyardCardTarget struct {
	BaseTarget
}

func TargetCardInYourGraveyard() Target {
	return &GraveyardCardTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *GraveyardCardTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	var result []uuid.UUID
	for _, c := range p.Graveyard() {
		result = append(result, c.ID())
	}
	return result
}

func (t *GraveyardCardTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}

// OpponentTarget targets an opponent.
type OpponentTarget struct {
	BaseTarget
}

func TargetOpponent() Target {
	return &OpponentTarget{
		BaseTarget: BaseTarget{min: 1, max: 1},
	}
}

func (t *OpponentTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, p := range g.Players {
		if p.PlayerID() != controller {
			result = append(result, p.PlayerID())
		}
	}
	return result
}

func (t *OpponentTarget) Choose(controller uuid.UUID, _ Card, g *Game, chosen []uuid.UUID) error {
	t.chosen = chosen
	return nil
}
